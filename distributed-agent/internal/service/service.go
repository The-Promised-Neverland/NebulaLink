package service

import (
	"archive/tar"
	"io"
	"os"
	"path/filepath"

	"github.com/The-Promised-Neverland/agent/internal/config"
	"github.com/The-Promised-Neverland/agent/internal/models"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
)

type Service struct {
	cfg *config.Config
}

func NewService(cfg *config.Config) *Service {
	return &Service{
		cfg: cfg,
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

func (s *Service) StreamRequestedFileSystem(path string) (<-chan []byte, <-chan error) {
	dataCh := make(chan []byte, 8)
	errCh := make(chan error, 1)

	go func() {
		defer close(dataCh)
		defer close(errCh)

		pr, pw := io.Pipe()
		tw := tar.NewWriter(pw)
		defer tw.Close()

		go func() {
			defer pw.Close()
			if walkErr := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				header, err := tar.FileInfoHeader(info, "")
				if err != nil {
					return err
				}
				rel, _ := filepath.Rel(path, filePath)
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
