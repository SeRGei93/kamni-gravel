package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var unsafeFileNameChars = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

// LocalFileStorage сохраняет файлы в локальную директорию, примонтированную как общий volume.
type LocalFileStorage struct {
	root string
}

// NewLocalFileStorage создаёт локальное файловое хранилище.
func NewLocalFileStorage(root string) *LocalFileStorage {
	root = strings.TrimSpace(root)
	if root == "" {
		root = "storage"
	}

	return &LocalFileStorage{root: root}
}

// SaveEventFile сохраняет файл события и возвращает путь относительно корня хранилища.
func (s *LocalFileStorage) SaveEventFile(ctx context.Context, eventID uint, originalName string, src io.Reader) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}

	fileName := sanitizeFileName(originalName)
	if fileName == "" {
		fileName = "event-file.gpx"
	}

	relativeDir := filepath.Join("events", fmt.Sprint(eventID), "gpx")
	absoluteDir := filepath.Join(s.root, relativeDir)
	if err := os.MkdirAll(absoluteDir, 0o755); err != nil {
		return "", fmt.Errorf("create event file directory: %w", err)
	}

	relativePath := filepath.Join(
		relativeDir,
		fmt.Sprintf("%s_%s", time.Now().Format("02-01-2006_15-04-05"), fileName),
	)
	absolutePath := filepath.Join(s.root, relativePath)

	dst, err := os.OpenFile(absolutePath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return "", fmt.Errorf("create event file: %w", err)
	}

	_, copyErr := io.Copy(dst, src)
	closeErr := dst.Close()
	if copyErr != nil {
		_ = os.Remove(absolutePath)
		return "", fmt.Errorf("save event file: %w", copyErr)
	}
	if closeErr != nil {
		_ = os.Remove(absolutePath)
		return "", fmt.Errorf("close event file: %w", closeErr)
	}
	if err := ctx.Err(); err != nil {
		_ = os.Remove(absolutePath)
		return "", err
	}

	return filepath.ToSlash(relativePath), nil
}

func sanitizeFileName(name string) string {
	base := filepath.Base(strings.TrimSpace(name))
	if base == "." || base == string(filepath.Separator) {
		return ""
	}

	base = strings.Trim(base, ".")
	base = strings.ReplaceAll(base, " ", "_")
	base = unsafeFileNameChars.ReplaceAllString(base, "_")
	base = strings.Trim(base, "_")
	if base == "" {
		return ""
	}

	if len(base) <= 120 {
		return base
	}

	ext := filepath.Ext(base)
	stem := strings.TrimSuffix(base, ext)
	maxStemLen := 120 - len(ext)
	if maxStemLen < 1 {
		return base[:120]
	}

	return stem[:maxStemLen] + ext
}
