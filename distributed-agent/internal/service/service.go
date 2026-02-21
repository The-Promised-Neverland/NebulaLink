package service

import (
	"archive/tar"
	"errors"
	"io"
	"os"
	"path/filepath"

	stun "github.com/The-Promised-Neverland/agent/internal/STUN"
	"github.com/The-Promised-Neverland/agent/internal/config"
	"github.com/The-Promised-Neverland/agent/internal/models"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
)

type Service struct {
	cfg        *config.Config
	stunClient *stun.STUNClient
}

func NewService(cfg *config.Config) *Service {
	stunClient := stun.NewSTUNserver(cfg)
	return &Service{
		cfg:        cfg,
		stunClient: stunClient,
	}
}

func (s *Service) GetHostMetrics() *models.HostMetrics {
	cpuPercent, _ := cpu.Percent(0, false)
	memStat, _ := mem.VirtualMemory()
	diskStat, _ := disk.Usage("/")
	hostInfo, _ := host.Info()

	return &models.HostMetrics{
		CPUUsage:    cpuPercent[0],
		MemoryUsage: memStat.UsedPercent,
		DiskUsage:   diskStat.UsedPercent,
		Hostname:    hostInfo.Hostname,
		OS:          hostInfo.OS,
		Uptime:      hostInfo.Uptime,
	}
}

func (s *Service) GetSTUNEndpoint() string {
	return s.stunClient.GetCurrentEndpoint()
}

func (s *Service) GetSTUNClient() *stun.STUNClient {
	return s.stunClient
}

func (s *Service) StreamRequestedFileSystem(path string) (<-chan []byte, <-chan error) {
	dataCh := make(chan []byte, 8)
	errCh := make(chan error, 1)
	sharedPath, err := s.cfg.SharedFolderPath()
	if err != nil {
		errCh <- errors.New("Shared path not provided")
		close(errCh)
		return nil, errCh
	}
	targetPath := filepath.Join(sharedPath, path)
	targetPath = filepath.Clean(targetPath)

	go func() {
		defer close(dataCh)
		defer close(errCh)

		pr, pw := io.Pipe()
		tw := tar.NewWriter(pw)
		defer tw.Close()

		go func() {
			defer pw.Close()
			// Get the base path for relative path calculation
			basePath := targetPath
			if info, err := os.Stat(targetPath); err == nil && !info.IsDir() {
				// If targetPath is a file, use its parent directory as base
				basePath = filepath.Dir(targetPath)
			}

			if walkErr := filepath.Walk(targetPath, func(filePath string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				// Calculate relative path from basePath
				rel, err := filepath.Rel(basePath, filePath)
				if err != nil {
					return err
				}
				// Skip if relative path is "." or empty (root directory entry)
				if rel == "." || rel == "" {
					return nil
				}
				header, err := tar.FileInfoHeader(info, "")
				if err != nil {
					return err
				}
				header.Name = filepath.ToSlash(rel)
				if err := tw.WriteHeader(header); err != nil {
					return err
				}
				if !info.IsDir() {
					f, err := os.Open(filePath)
					if err != nil {
						return err
					}
					defer f.Close()
					if _, err := io.Copy(tw, f); err != nil {
						return err
					}
				}
				return nil
			}); walkErr != nil {
				_ = pw.CloseWithError(walkErr)
			}
		}()

		buf := make([]byte, 64*1024)
		for {
			n, err := pr.Read(buf)
			if n > 0 {
				chunk := make([]byte, n)
				copy(chunk, buf[:n])
				dataCh <- chunk
			}
			if err == io.EOF {
				break
			}
			if err != nil {
				errCh <- err
				break
			}
		}
	}()

	return dataCh, errCh
}
