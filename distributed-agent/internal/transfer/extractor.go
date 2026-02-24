package transfer

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/The-Promised-Neverland/agent/internal/config"
	"github.com/The-Promised-Neverland/agent/pkg/logger"
)

type TarExtractor struct {
	config *config.Config
}

func NewTarExtractor(cfg *config.Config) *TarExtractor {
	return &TarExtractor{
		config: cfg,
	}
}

func (e *TarExtractor) ExtractTar(tarPath string, sourceAgentID string) error {
	sharedPath, err := e.config.SharedFolderPath()
	if err != nil {
		return fmt.Errorf("failed to get shared folder path: %w", err)
	}
	extractPath := filepath.Join(sharedPath, "transfers", sourceAgentID)
	if err := os.MkdirAll(extractPath, 0755); err != nil {
		return fmt.Errorf("failed to create extract directory: %w", err)
	}
	tarFile, err := os.Open(tarPath)
	if err != nil {
		return fmt.Errorf("failed to open tar file: %w", err)
	}
	defer tarFile.Close()
	tarReader := tar.NewReader(tarFile)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar header: %w", err)
		}
		cleanName := filepath.Clean(header.Name)
		if cleanName == "" || cleanName == "." || cleanName == "/" {
			logger.Log.Error("Skipping root/empty tar entry", "name", header.Name)
			continue
		}
		targetPath := filepath.Join(extractPath, cleanName)
		extractPathClean := filepath.Clean(extractPath)
		if !filepath.HasPrefix(targetPath, extractPathClean+string(os.PathSeparator)) && targetPath != extractPathClean {
			logger.Log.Warn("Skipping file with invalid path (outside extract directory)",
				"path", header.Name,
				"targetPath", targetPath)
			continue
		}
		if targetPath == extractPathClean {
			logger.Log.Debug("Skipping entry that would overwrite extract directory", "name", header.Name)
			continue
		}
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}
			logger.Log.Info("Extracted directory", "path", targetPath)

		case tar.TypeReg:
			if info, err := os.Stat(targetPath); err == nil && info.IsDir() {
				logger.Log.Warn("Skipping file entry - target path is a directory",
					"path", targetPath,
					"headerName", header.Name)
				continue
			}
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return fmt.Errorf("failed to create parent directory: %w", err)
			}
			outFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.FileMode(header.Mode))
			if err != nil {
				return fmt.Errorf("failed to create file: %w", err)
			}
			if _, err := io.Copy(outFile, tarReader); err != nil {
				outFile.Close()
				return fmt.Errorf("failed to write file: %w", err)
			}
			outFile.Close()
			logger.Log.Debug("Extracted file", "path", targetPath, "size", header.Size)
		default:
			logger.Log.Warn("Unsupported tar entry type", "type", header.Typeflag, "name", header.Name)
		}
	}
	logger.Log.Info("Successfully extracted tar to shared folder",
		"sourceAgent", sourceAgentID,
		"extractPath", extractPath)
	return nil
}



