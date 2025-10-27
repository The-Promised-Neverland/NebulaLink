package service

import (
	"github.com/The-Promised-Neverland/agent/internal/models"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
)

type OSManager interface {
	Restart() error
	Uninstall() error
}

type Service struct {
	osMgr OSManager
}

func NewService(osMgr OSManager) *Service {
	return &Service{
		osMgr: osMgr,
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

func (s *Service) RestartAgent() error {
	return s.osMgr.Restart()
}

func (s *Service) UninstallAgent(initiator string) error {
	return s.osMgr.Uninstall()
}
