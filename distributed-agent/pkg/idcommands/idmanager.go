package idcommands

import (
	"crypto/sha256"
	"fmt"
	"strings"

	"github.com/The-Promised-Neverland/agent/pkg/utils"
)

// GenerateAgentID creates a unique, persistent hardware-based ID
// Get Machine GUID (most reliable - from Windows registry)
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

// getMachineGUID retrieves Windows Machine GUID from registry
func getMachineGUID() (string, error) {
	output, err := utils.RunCommand(
        "powershell", "-NoProfile", "-Command",
        `(Get-ItemProperty 'HKLM:\SOFTWARE\Microsoft\Cryptography' -Name MachineGuid).MachineGuid`,
    )
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
	output, err := utils.RunCommand(
        "powershell", "-NoProfile", "-Command",
        `(Get-NetAdapter -Physical | Where-Object {$_.Status -eq 'Up'} | Select-Object -First 1).MacAddress`,
    )
    if err != nil {
        return "", fmt.Errorf("failed to get MAC address: %w", err)
    }
    macAddr := strings.TrimSpace(output)
    if macAddr == "" {
        return "", fmt.Errorf("no active network adapter found")
    }
    return macAddr, nil
}