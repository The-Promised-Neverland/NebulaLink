package policy

import (
	"fmt"
	"runtime"

	"github.com/The-Promised-Neverland/agent/internal/config"
)

type ServicePolicy interface {
	ConfigureAutoStart() error
	ConfigureRestartPolicy() error
}

func NewServicePolicy(cfg *config.Config) (ServicePolicy, error) {
	switch runtime.GOOS {
	case "windows":
		return NewWindowsPolicy(cfg), nil
	case "linux":
		return NewLinuxPolicy(cfg), nil
	case "darwin":
		return NewDarwinPolicy(cfg), nil
	default:
		return nil, fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
}
