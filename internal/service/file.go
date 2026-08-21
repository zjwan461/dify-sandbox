package service

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/langgenius/dify-sandbox/internal/static"
	"github.com/langgenius/dify-sandbox/internal/types"
)

// UploadFileResponse is the response for file upload
type UploadFileResponse struct {
	Filename string `json:"filename"`
	Size     int64  `json:"size"`
}

// UploadFile saves an uploaded file to the uploads directory
func UploadFile(file io.Reader, filename string, size int64) (*types.DifySandboxResponse, error) {
	uploadDir := static.GetDifySandboxGlobalConfigurations().UploadDir

	// Ensure upload directory exists
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return types.ErrorResponse(-500, fmt.Sprintf("failed to create upload directory: %v", err)), nil
	}

	// Generate unique filename to avoid conflicts
	ext := filepath.Ext(filename)
	baseName := strings.TrimSuffix(filename, ext)
	uniqueName := fmt.Sprintf("%s_%s%s", baseName, uuid.New().String()[:8], ext)

	// Sanitize filename to prevent path traversal
	uniqueName = filepath.Base(uniqueName)
	if uniqueName == "." || uniqueName == ".." || strings.Contains(uniqueName, "/") {
		return types.ErrorResponse(-400, "invalid filename"), nil
	}

	filePath := filepath.Join(uploadDir, uniqueName)

	// Create the file
	dst, err := os.Create(filePath)
	if err != nil {
		return types.ErrorResponse(-500, fmt.Sprintf("failed to create file: %v", err)), nil
	}
	defer dst.Close()

	// Copy the file content
	if _, err := io.Copy(dst, file); err != nil {
		return types.ErrorResponse(-500, fmt.Sprintf("failed to save file: %v", err)), nil
	}

	return types.SuccessResponse(&UploadFileResponse{
		Filename: uniqueName,
		Size:     size,
	}), nil
}

// DownloadFile returns the file path for downloading
func DownloadFile(filename string) (string, error) {
	// Sanitize filename to prevent path traversal
	filename = filepath.Base(filename)
	if filename == "." || filename == ".." || strings.Contains(filename, "/") {
		return "", fmt.Errorf("invalid filename")
	}

	uploadDir := static.GetDifySandboxGlobalConfigurations().UploadDir
	filePath := filepath.Join(uploadDir, filename)

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return "", fmt.Errorf("file not found")
	}

	return filePath, nil
}
