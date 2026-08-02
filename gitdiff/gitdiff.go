// Package gitdiff creates snapshots from revisions in isolated temporary worktrees.
package gitdiff

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/magnexis/taglock/config"
	"github.com/magnexis/taglock/engine"
	"github.com/magnexis/taglock/snapshot"
)

type Options struct {
	Patterns     []string
	Semantics    string
	Reproducible bool
	BuildTags    []string
}

func RepositoryRoot(ctx context.Context, dir string) (string, error) {
	command := exec.CommandContext(ctx, "git", "rev-parse", "--show-toplevel")
	command.Dir = dir
	output, err := command.Output()
	if err != nil {
		return "", fmt.Errorf("not a Git repository or Git unavailable: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}
func SnapshotRevision(ctx context.Context, repository, revision string, cfg config.Config, options Options) (document snapshot.Snapshot, err error) {
	if strings.TrimSpace(revision) == "" {
		return snapshot.Snapshot{}, fmt.Errorf("revision cannot be empty")
	}
	if err := verifyRevision(ctx, repository, revision); err != nil {
		return snapshot.Snapshot{}, err
	}
	temporary, err := os.MkdirTemp("", "taglock-worktree-")
	if err != nil {
		return snapshot.Snapshot{}, err
	}
	defer os.RemoveAll(temporary)
	added := false
	defer func() {
		if added {
			cleanup := exec.CommandContext(context.Background(), "git", "worktree", "remove", "--force", temporary)
			cleanup.Dir = repository
			if cleanupErr := cleanup.Run(); err == nil && cleanupErr != nil {
				err = fmt.Errorf("remove temporary worktree: %w", cleanupErr)
			}
		}
	}()
	command := exec.CommandContext(ctx, "git", "worktree", "add", "--detach", temporary, revision)
	command.Dir = repository
	if output, commandErr := command.CombinedOutput(); commandErr != nil {
		return snapshot.Snapshot{}, fmt.Errorf("create worktree for %s: %w: %s", revision, commandErr, strings.TrimSpace(string(output)))
	}
	added = true
	result, err := engine.AnalyzeWithOptions(ctx, options.Patterns, cfg, engine.Options{Dir: temporary, BuildTags: options.BuildTags})
	if err != nil {
		return snapshot.Snapshot{}, fmt.Errorf("analyze revision %s: %w", revision, err)
	}
	return snapshot.Build(result, cfg, snapshot.BuildOptions{Semantics: options.Semantics, Reproducible: options.Reproducible})
}
func verifyRevision(ctx context.Context, repository, revision string) error {
	command := exec.CommandContext(ctx, "git", "rev-parse", "--verify", revision+"^{commit}")
	command.Dir = repository
	if output, err := command.CombinedOutput(); err != nil {
		return fmt.Errorf("revision %q is unavailable (the clone may be shallow); fetch it explicitly before running TagLock: %s", revision, strings.TrimSpace(string(output)))
	}
	return nil
}
func CompareRevisions(ctx context.Context, dir, base, head string, cfg config.Config, options Options) (snapshot.Snapshot, snapshot.Snapshot, error) {
	root, err := RepositoryRoot(ctx, dir)
	if err != nil {
		return snapshot.Snapshot{}, snapshot.Snapshot{}, err
	}
	before, err := SnapshotRevision(ctx, root, base, cfg, options)
	if err != nil {
		return snapshot.Snapshot{}, snapshot.Snapshot{}, err
	}
	after, err := SnapshotRevision(ctx, root, head, cfg, options)
	if err != nil {
		return snapshot.Snapshot{}, snapshot.Snapshot{}, err
	}
	return before, after, nil
}
func IsSnapshot(value string) bool {
	info, err := os.Stat(value)
	return err == nil && !info.IsDir() && strings.EqualFold(filepath.Ext(value), ".json")
}
