package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGeneratedCodeIsCurrent(t *testing.T) {
	t.Chdir("../..")
	if err := generate(true); err != nil {
		t.Fatal(err)
	}
}

func TestCompareDirectoriesDetectsStaleFiles(t *testing.T) {
	for _, change := range []string{"changed", "missing", "extra"} {
		t.Run(change, func(t *testing.T) {
			expected, actual := t.TempDir(), t.TempDir()
			write := func(directory, name, content string) {
				t.Helper()
				if err := os.WriteFile(filepath.Join(directory, name), []byte(content), 0600); err != nil {
					t.Fatal(err)
				}
			}
			write(expected, "generated.go", "expected")
			write(actual, "generated.go", "expected")
			if err := compareDirectories(expected, actual); err != nil {
				t.Fatal(err)
			}
			switch change {
			case "changed":
				write(actual, "generated.go", "stale")
			case "missing":
				if err := os.Remove(filepath.Join(actual, "generated.go")); err != nil {
					t.Fatal(err)
				}
			case "extra":
				write(actual, "extra.go", "stale")
			}
			if err := compareDirectories(expected, actual); err == nil {
				t.Fatal("stale output accepted")
			}
		})
	}
}
