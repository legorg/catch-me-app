// Command apigen generates the API document, or checks it without modifying files.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

const swagVersion = "v1.16.6"

func main() {
	check := flag.Bool("check", false, "Fail if api/openapi.json is out of date, without modifying it")
	flag.Parse()
	if err := generate(*check); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func generate(check bool) error {
	dir, err := os.MkdirTemp("", "catch-me-openapi-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)

	goBinary := os.Getenv("GO_BINARY")
	if goBinary == "" {
		goBinary = "go"
	}
	cmd := exec.Command(goBinary, "run", "github.com/swaggo/swag/cmd/swag@"+swagVersion,
		"init", "-g", "main.go", "-d", "cmd/server,internal/api", "--parseInternal",
		"--requiredByDefault", "--outputTypes", "json", "--output", dir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("generate API document: %w", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "swagger.json"))
	if err != nil {
		return err
	}
	var document map[string]any
	if err := json.Unmarshal(data, &document); err != nil {
		return err
	}
	document["x-generated-by"] = "Swaggo " + swagVersion + "; do not edit, run npm run api:update from app/"
	data, err = json.MarshalIndent(document, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	path := filepath.Join("api", "openapi.json")
	if check {
		current, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if !bytes.Equal(current, data) {
			return fmt.Errorf("%s is stale; run npm run api:update from app/", path)
		}
		fmt.Println("API document matches Go source.")
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
