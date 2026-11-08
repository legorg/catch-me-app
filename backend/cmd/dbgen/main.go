// Command dbgen generates Ent code or checks it without modifying repository files.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"

	"entgo.io/ent/entc"
	"entgo.io/ent/entc/gen"
)

const outputPath = "internal/db/ent"

func main() {
	check := flag.Bool("check", false, "Check generated Ent code without modifying it")
	flag.Parse()
	if err := generate(*check); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func generate(check bool) error {
	output := outputPath
	if check {
		temporary, err := os.MkdirTemp("", "catch-me-ent-*")
		if err != nil {
			return err
		}
		defer os.RemoveAll(temporary)
		output = temporary
	}
	// Ent's version is pinned by go.mod; Package stays stable in temporary output.
	if err := entc.Generate("./"+outputPath+"/schema", &gen.Config{
		Target: output, Package: "catch-me/backend/internal/db/ent",
	}); err != nil {
		return fmt.Errorf("generate Ent code: %w", err)
	}
	if check {
		if err := compareDirectories(output, outputPath); err != nil {
			return err
		}
		fmt.Println("Ent code matches Go schemas.")
	} else {
		fmt.Println("Generated Ent code from Go schemas.")
	}
	return nil
}

func generatedFiles(directory string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(directory, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(directory, path)
		if err != nil {
			return err
		}
		// schema/ is handwritten input, not generated output.
		if relative == "schema" && entry.IsDir() {
			return filepath.SkipDir
		}
		if !entry.IsDir() {
			files = append(files, relative)
		}
		return nil
	})
	slices.Sort(files)
	return files, err
}

func compareDirectories(expected, actual string) error {
	expectedFiles, err := generatedFiles(expected)
	if err != nil {
		return err
	}
	actualFiles, err := generatedFiles(actual)
	if err != nil {
		return err
	}
	if !slices.Equal(expectedFiles, actualFiles) {
		return fmt.Errorf("Ent files are stale; run go run ./cmd/dbgen from backend/")
	}
	for _, file := range expectedFiles {
		want, err := os.ReadFile(filepath.Join(expected, file))
		if err != nil {
			return err
		}
		got, err := os.ReadFile(filepath.Join(actual, file))
		if err != nil {
			return err
		}
		if !bytes.Equal(want, got) {
			return fmt.Errorf("Ent file %s is stale; run go run ./cmd/dbgen from backend/", file)
		}
	}
	return nil
}
