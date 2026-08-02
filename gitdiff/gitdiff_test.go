package gitdiff_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/magnexis/taglock/config"
	"github.com/magnexis/taglock/gitdiff"
)

func TestCompareRevisionsPreservesDirtyWorktree(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git unavailable")
	}
	repository := t.TempDir()
	git(t, repository, "init", "-b", "main")
	git(t, repository, "config", "user.name", "TagLock Test")
	git(t, repository, "config", "user.email", "taglock@example.invalid")
	write(t, repository, "go.mod", "module example.test/contracts\n\ngo 1.24\n")
	write(t, repository, "model.go", "package contracts\n\ntype User struct { ID string `json:\"id\"` }\n")
	git(t, repository, "add", "go.mod", "model.go")
	git(t, repository, "commit", "-m", "base")
	base := strings.TrimSpace(git(t, repository, "rev-parse", "HEAD"))
	write(t, repository, "model.go", "package contracts\n\ntype User struct { ID int `json:\"id\"` }\n")
	git(t, repository, "add", "model.go")
	git(t, repository, "commit", "-m", "head")
	head := strings.TrimSpace(git(t, repository, "rev-parse", "HEAD"))
	dirty := filepath.Join(repository, "local-notes.txt")
	if err := os.WriteFile(dirty, []byte("keep me"), 0o644); err != nil {
		t.Fatal(err)
	}
	before, after, err := gitdiff.CompareRevisions(context.Background(), repository, base, head, config.Default(), gitdiff.Options{Patterns: []string{"./..."}, Semantics: "v1", Reproducible: true})
	if err != nil {
		t.Fatal(err)
	}
	if before.Fingerprint == after.Fingerprint {
		t.Fatal("revision snapshots unexpectedly match")
	}
	data, err := os.ReadFile(dirty)
	if err != nil || string(data) != "keep me" {
		t.Fatalf("dirty worktree changed: %q %v", data, err)
	}
}

func TestCompareRevisionsRejectsMissingHistory(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git unavailable")
	}
	repository := t.TempDir()
	git(t, repository, "init", "-b", "main")
	git(t, repository, "config", "user.name", "TagLock Test")
	git(t, repository, "config", "user.email", "taglock@example.invalid")
	write(t, repository, "go.mod", "module example.test/contracts\n\ngo 1.24\n")
	write(t, repository, "model.go", "package contracts\n\ntype User struct { ID string `json:\"id\"` }\n")
	git(t, repository, "add", "go.mod", "model.go")
	git(t, repository, "commit", "-m", "base")
	_, _, err := gitdiff.CompareRevisions(context.Background(), repository, "missing-revision", "HEAD", config.Default(), gitdiff.Options{Patterns: []string{"./..."}, Semantics: "v1", Reproducible: true})
	if err == nil || !strings.Contains(err.Error(), "missing-revision") {
		t.Fatalf("expected actionable missing revision error, got %v", err)
	}
}

func BenchmarkGitSnapshotOrchestration(b *testing.B) {
	if _, err := exec.LookPath("git"); err != nil {
		b.Skip("git unavailable")
	}
	repository := b.TempDir()
	git(b, repository, "init", "-b", "main")
	git(b, repository, "config", "user.name", "TagLock Benchmark")
	git(b, repository, "config", "user.email", "taglock@example.invalid")
	write(b, repository, "go.mod", "module example.test/contracts\n\ngo 1.24\n")
	write(b, repository, "model.go", "package contracts\n\ntype User struct { ID string `json:\"id\"` }\n")
	git(b, repository, "add", "go.mod", "model.go")
	git(b, repository, "commit", "-m", "base")
	b.ResetTimer()
	for range b.N {
		if _, _, err := gitdiff.CompareRevisions(context.Background(), repository, "HEAD", "HEAD", config.Default(), gitdiff.Options{Patterns: []string{"./..."}, Semantics: "v1", Reproducible: true}); err != nil {
			b.Fatal(err)
		}
	}
}

func git(t testing.TB, dir string, args ...string) string {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = dir
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, output)
	}
	return string(output)
}
func write(t testing.TB, dir, name, value string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(value), 0o644); err != nil {
		t.Fatal(err)
	}
}
