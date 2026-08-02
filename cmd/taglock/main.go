package main

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/theworker02/taglock/analyzer"
	"github.com/theworker02/taglock/internal/cli"
	"golang.org/x/tools/go/analysis/unitchecker"
)

func main() {
	if len(os.Args) == 2 && os.Args[1] == "-V=full" {
		printVetVersion()
		return
	}
	if vetInvocation(os.Args[1:]) {
		unitchecker.Main(analyzer.Analyzer)
		return
	}
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr))
}

func vetInvocation(arguments []string) bool {
	for _, argument := range arguments {
		if argument == "-flags" || strings.HasPrefix(argument, "-V=") || strings.HasPrefix(argument, "-c=") || strings.HasSuffix(argument, ".cfg") {
			return true
		}
	}
	return false
}

// printVetVersion emits a content-derived version understood by the go command's
// vet-tool protocol. x/tools v0.30's development-version placeholder is rejected
// by newer Go toolchains, so TagLock supplies its own deterministic cache key.
func printVetVersion() {
	executable, err := os.Executable()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	file, err := os.Open(executable)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer file.Close()
	digest := sha256.New()
	if _, err := io.Copy(digest, file); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("taglock version taglock-%x\n", digest.Sum(nil))
}
