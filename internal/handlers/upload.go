package handlers

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

var allowedImageExts = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".webp": true,
	".gif":  true,
}

func saveUpload(r *http.Request, field, uploadDir string) (string, error) {
	file, header, err := r.FormFile(field)
	if err != nil {
		// No file uploaded
		return "", nil
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if !allowedImageExts[ext] {
		return "", fmt.Errorf("invalid file type: %s", ext)
	}

	filename := uuid.New().String() + ext
	dst, err := os.Create(filepath.Join(uploadDir, filename))
	if err != nil {
		return "", fmt.Errorf("create upload file: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		return "", fmt.Errorf("write upload file: %w", err)
	}

	return filename, nil
}

func deleteUpload(uploadDir, filename string) {
	if filename == "" {
		return
	}
	path := filepath.Join(uploadDir, filename)
	os.Remove(path)
}
