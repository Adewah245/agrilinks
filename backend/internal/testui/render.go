// Package testui validates the React UI against actual Go page data.
package testui

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func HTML(t *testing.T, shell string) string {
	t.Helper()
	var renderer string
	for _, root := range []string{"../../../frontend", "../../frontend", "frontend"} {
		candidate := filepath.Join(root, ".test-build", "render.mjs")
		if _, err := os.Stat(candidate); err == nil {
			renderer = candidate
			break
		}
	}
	if renderer == "" {
		t.Fatal("React test renderer missing: run npm --prefix frontend run build:test")
	}
	if directory := os.Getenv("REACT_PREVIEW_DIR"); directory != "" {
		if err := os.MkdirAll(directory, 0755); err != nil {
			t.Fatal(err)
		}
		name := regexp.MustCompile(`[^a-zA-Z0-9_-]+`).ReplaceAllString(t.Name(), "-")
		if err := os.WriteFile(filepath.Join(directory, name+".html"), []byte(shell), 0644); err != nil {
			t.Fatal(err)
		}
	}
	command := exec.Command("node", renderer)
	command.Stdin = strings.NewReader(shell)
	var stderr bytes.Buffer
	command.Stderr = &stderr
	output, err := command.Output()
	if err != nil {
		t.Fatalf("React render failed: %v\n%s", err, stderr.String())
	}
	if stderr.Len() > 0 {
		t.Fatalf("React render warning: %s", stderr.String())
	}
	return string(output)
}
