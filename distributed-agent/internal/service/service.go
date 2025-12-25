package service

import (
	"github.com/The-Promised-Neverland/agent/internal/models"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
)

type Service struct{}

func NewService() *Service {
	return &Service{}
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
