package storage

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestLocalFileStorageSaveEventFile(t *testing.T) {
	root := t.TempDir()
	storage := NewLocalFileStorage(root)

	relativePath, err := storage.SaveEventFile(
		context.Background(),
		42,
		"../../My Route.gpx",
		strings.NewReader("<gpx></gpx>"),
	)
	if err != nil {
		t.Fatalf("save event file: %v", err)
	}

	if !strings.HasPrefix(relativePath, "events/42/gpx/") {
		t.Fatalf("relative path prefix mismatch: %q", relativePath)
	}
	if strings.Contains(relativePath, "..") {
		t.Fatalf("relative path should be sanitized: %q", relativePath)
	}
	fileName := filepath.Base(relativePath)
	matched, err := regexp.MatchString(`^\d{2}-\d{2}-\d{4}_\d{2}-\d{2}-\d{2}_My_Route\.gpx$`, fileName)
	if err != nil {
		t.Fatalf("compile file name regexp: %v", err)
	}
	if !matched {
		t.Fatalf("file name format mismatch: %q", fileName)
	}

	content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relativePath)))
	if err != nil {
		t.Fatalf("read saved file: %v", err)
	}
	if string(content) != "<gpx></gpx>" {
		t.Fatalf("saved content mismatch: %q", string(content))
	}
}
