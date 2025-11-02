package idcommands

import (
	"crypto/sha256"
	"fmt"
	"runtime"
	"strings"

	"github.com/The-Promised-Neverland/agent/pkg/utils"
)

// GenerateAgentID creates a unique, persistent hardware-based ID
func GenerateAgentID() string {
	machineGUID, err := getMachineGUID()
	if err != nil {
		return ""
	}
	macAddr, err := getPrimaryMACAddress()
	if err != nil {
		macAddr = "no-network"
	}
	combined := fmt.Sprintf("%s:%s", machineGUID, macAddr)
	hash := sha256.Sum256([]byte(combined))
	agentID := fmt.Sprintf("%x", hash)
	return agentID
}

// getMachineGUID retrieves machine GUID based on OS
func getMachineGUID() (string, error) {
	var output string
	var err error

	switch runtime.GOOS {
	case "windows":
		output, err = utils.RunCommand(
			"powershell", "-NoProfile", "-Command",
			`(Get-ItemProperty 'HKLM:\SOFTWARE\Microsoft\Cryptography' -Name MachineGuid).MachineGuid`,
		)
	case "linux":
		// Try multiple sources in order of reliability
		// 1. Try /etc/machine-id (systemd)
		output, err = utils.RunCommand("cat", "/etc/machine-id")
		if err != nil || strings.TrimSpace(output) == "" {
			// 2. Try /var/lib/dbus/machine-id (dbus)
			output, err = utils.RunCommand("cat", "/var/lib/dbus/machine-id")
		}
		if err != nil || strings.TrimSpace(output) == "" {
			// 3. Fallback to DMI product UUID
			output, err = utils.RunCommand("cat", "/sys/class/dmi/id/product_uuid")
		}
	case "darwin": // macOS
		output, err = utils.RunCommand("ioreg", "-rd1", "-c", "IOPlatformExpertDevice")
		if err == nil {
			// Extract UUID from output
			lines := strings.Split(output, "\n")
			for _, line := range lines {
				if strings.Contains(line, "IOPlatformUUID") {
					parts := strings.Split(line, "\"")
					if len(parts) >= 4 {
						output = parts[3]
						break
					}
				}
			}
		}
	default:
		return "", fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}

	if err != nil {
		return "", fmt.Errorf("failed to query machine GUID: %w", err)
	}

	machineGUID := strings.TrimSpace(output)
	if machineGUID == "" {
		return "", fmt.Errorf("machine GUID is empty")
	}

	return machineGUID, nil
}

// getPrimaryMACAddress retrieves the MAC address of the primary network adapter
func getPrimaryMACAddress() (string, error) {
	var output string
	var err error

	switch runtime.GOOS {
	case "windows":
		output, err = utils.RunCommand(
			"powershell", "-NoProfile", "-Command",
			`(Get-NetAdapter -Physical | Where-Object {$_.Status -eq 'Up'} | Select-Object -First 1).MacAddress`,
		)
	case "linux":
		// Get the default route interface
		routeOutput, routeErr := utils.RunCommand("sh", "-c", "ip route | grep default | awk '{print $5}' | head -n1")
		if routeErr != nil || strings.TrimSpace(routeOutput) == "" {
			// Fallback: get first non-loopback interface
			output, err = utils.RunCommand("sh", "-c", "cat /sys/class/net/$(ls /sys/class/net | grep -v lo | head -n1)/address")
		} else {
			iface := strings.TrimSpace(routeOutput)
			output, err = utils.RunCommand("cat", fmt.Sprintf("/sys/class/net/%s/address", iface))
		}
	case "darwin": // macOS
		output, err = utils.RunCommand("sh", "-c", "ifconfig | grep -A 5 'status: active' | grep ether | head -n1 | awk '{print $2}'")
	default:
		return "", fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}

	if err != nil {
		return "", fmt.Errorf("failed to get MAC address: %w", err)
	}

	macAddr := strings.TrimSpace(output)
	if macAddr == "" {
		return "", fmt.Errorf("no active network adapter found")
	}

	return macAddr, nil
}