package policy

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/The-Promised-Neverland/agent/internal/config"
	"github.com/The-Promised-Neverland/agent/pkg/logger"
	"github.com/The-Promised-Neverland/agent/pkg/utils"
)

type DarwinPolicy struct {
	serviceName string
	binaryPath  string
}

func NewDarwinPolicy(cfg *config.Config) *DarwinPolicy {
	return &DarwinPolicy{
		serviceName: cfg.ServiceName(), 
		binaryPath:  cfg.BinaryPath(),  
	}
}

func (p *DarwinPolicy) ConfigureAutoStart() error {
	plistPath := p.plistPath()
	if err := os.MkdirAll(filepath.Dir(plistPath), 0755); err != nil {
		return err
	}
	content := p.plistContent()
	if err := os.WriteFile(plistPath, []byte(content), 0644); err != nil {
		return err
	}
	_, _ = utils.RunCommand("launchctl", "bootout", "system", plistPath)
	if _, err := utils.RunCommand("launchctl", "bootstrap", "system", plistPath); err != nil {
		return err
	}
	logger.Log.Info("launchd plist installed and loaded", "path", plistPath)
	return nil
}

func (p *DarwinPolicy) ConfigureRestartPolicy() error {
	logger.Log.Info("launchd restart policy enforced via KeepAlive")
	return nil
}

func (p *DarwinPolicy) plistPath() string {
	return filepath.Join(
		"/Library/LaunchDaemons",
		p.serviceName+".plist",
	)
}

func (p *DarwinPolicy) plistContent() string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN"
 "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>%s</string>

	<key>ProgramArguments</key>
	<array>
		<string>%s</string>
	</array>

	<key>RunAtLoad</key>
	<true/>

	<key>KeepAlive</key>
	<true/>

	<key>ProcessType</key>
	<string>Background</string>

	<key>StandardOutPath</key>
	<string>/var/log/%s.out</string>

	<key>StandardErrorPath</key>
	<string>/var/log/%s.err</string>
</dict>
</plist>
`,
		p.serviceName,
		p.binaryPath,
		sanitizeLabel(p.serviceName),
		sanitizeLabel(p.serviceName),
	)
}

func sanitizeLabel(label string) string {
	return strings.ReplaceAll(label, ".", "_")
}
