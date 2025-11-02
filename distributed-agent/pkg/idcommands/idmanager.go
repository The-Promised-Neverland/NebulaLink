package idcommands

import (
	"crypto/sha256"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"runtime"
	"strings"

	"github.com/The-Promised-Neverland/agent/pkg/utils"
)

// GenerateAgentID creates a unique, persistent hardware-based ID
func GenerateAgentID() string {
	machineGUID, err := getMachineGUID()
	if err != nil {
		log.Printf("⚠️ Failed to get machine GUID: %v", err)
		// Use hostname as fallback
		machineGUID, _ = getHostname()
		if machineGUID == "" {
			machineGUID = "unknown-machine"
		}
	}
	
	macAddr, err := getPrimaryMACAddress()
	if err != nil {
		log.Printf("⚠️ Failed to get MAC address: %v", err)
		macAddr = "no-network"
	}
	
	combined := fmt.Sprintf("%s:%s", machineGUID, macAddr)
	hash := sha256.Sum256([]byte(combined))
	agentID := fmt.Sprintf("%x", hash)
	
	log.Printf("✅ Generated Agent ID: %s (from %s:%s)", agentID[:16], machineGUID, macAddr)
	return agentID
}

// getHostname gets the system hostname as a fallback
func getHostname() (string, error) {
	output, err := utils.RunCommand("hostname")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(output), nil
}

// getMachineGUID retrieves machine GUID based on OS
func getMachineGUID() (string, error) {
	switch runtime.GOOS {
	case "windows":
		return getMachineGUIDWindows()
	case "linux":
		return getMachineGUIDLinux()
	case "darwin":
		return getMachineGUIDDarwin()
	default:
		return "", fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

// getMachineGUIDWindows gets Windows machine GUID
func getMachineGUIDWindows() (string, error) {
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

// getMachineGUIDLinux gets Linux machine ID
func getMachineGUIDLinux() (string, error) {
	// Try reading files directly (more reliable than running commands)
	paths := []string{
		"/etc/machine-id",
		"/var/lib/dbus/machine-id",
	}
	
	for _, path := range paths {
		data, err := ioutil.ReadFile(path)
		if err == nil {
			machineID := strings.TrimSpace(string(data))
			if machineID != "" {
				log.Printf("✅ Found machine ID in %s", path)
				return machineID, nil
			}
		}
	}
	
	// Try DMI product UUID (requires root on some systems)
	data, err := ioutil.ReadFile("/sys/class/dmi/id/product_uuid")
	if err == nil {
		productUUID := strings.TrimSpace(string(data))
		if productUUID != "" && productUUID != "00000000-0000-0000-0000-000000000000" {
			log.Printf("✅ Found product UUID")
			return productUUID, nil
		}
	}
	
	// Try reading boot ID (changes on reboot, but better than nothing)
	data, err = ioutil.ReadFile("/proc/sys/kernel/random/boot_id")
	if err == nil {
		bootID := strings.TrimSpace(string(data))
		if bootID != "" {
			log.Printf("⚠️ Using boot_id (will change on reboot)")
			return bootID, nil
		}
	}
	
	// Last resort: use hostname
	hostname, err := getHostname()
	if err == nil && hostname != "" {
		log.Printf("⚠️ Using hostname as machine ID: %s", hostname)
		return hostname, nil
	}
	
	return "", fmt.Errorf("could not determine machine ID from any source")
}

// getMachineGUIDDarwin gets macOS hardware UUID
func getMachineGUIDDarwin() (string, error) {
	output, err := utils.RunCommand("ioreg", "-rd1", "-c", "IOPlatformExpertDevice")
	if err != nil {
		return "", fmt.Errorf("failed to query hardware UUID: %w", err)
	}
	
	// Extract UUID from output
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "IOPlatformUUID") {
			parts := strings.Split(line, "\"")
			if len(parts) >= 4 {
				return parts[3], nil
			}
		}
	}
	
	return "", fmt.Errorf("IOPlatformUUID not found in output")
}

// getPrimaryMACAddress retrieves the MAC address of the primary network adapter
func getPrimaryMACAddress() (string, error) {
	switch runtime.GOOS {
	case "windows":
		return getMACAddressWindows()
	case "linux":
		return getMACAddressLinux()
	case "darwin":
		return getMACAddressDarwin()
	default:
		return "", fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

// getMACAddressWindows gets Windows MAC address
func getMACAddressWindows() (string, error) {
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

// getMACAddressLinux gets Linux MAC address
func getMACAddressLinux() (string, error) {
	// Method 1: Use Go's net package (most reliable)
	interfaces, err := net.Interfaces()
	if err == nil {
		for _, iface := range interfaces {
			// Skip loopback and down interfaces
			if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
				continue
			}
			
			// Get MAC address
			if iface.HardwareAddr != nil && len(iface.HardwareAddr) > 0 {
				mac := iface.HardwareAddr.String()
				if mac != "" && mac != "00:00:00:00:00:00" {
					log.Printf("✅ Found MAC address from interface %s: %s", iface.Name, mac)
					return mac, nil
				}
			}
		}
	}
	
	// Method 2: Try to read from sysfs for default route interface
	routeOutput, err := utils.RunCommand("sh", "-c", "ip route show default | awk '{print $5}' | head -n1")
	if err == nil {
		iface := strings.TrimSpace(routeOutput)
		if iface != "" {
			path := fmt.Sprintf("/sys/class/net/%s/address", iface)
			data, err := ioutil.ReadFile(path)
			if err == nil {
				mac := strings.TrimSpace(string(data))
				if mac != "" && mac != "00:00:00:00:00:00" {
					log.Printf("✅ Found MAC address from default interface %s: %s", iface, mac)
					return mac, nil
				}
			}
		}
	}
	
	// Method 3: Get first non-loopback interface from sysfs
	output, err := utils.RunCommand("sh", "-c", "ls /sys/class/net | grep -v lo | head -n1")
	if err == nil {
		iface := strings.TrimSpace(output)
		if iface != "" {
			path := fmt.Sprintf("/sys/class/net/%s/address", iface)
			data, err := ioutil.ReadFile(path)
			if err == nil {
				mac := strings.TrimSpace(string(data))
				if mac != "" && mac != "00:00:00:00:00:00" {
					log.Printf("✅ Found MAC address from interface %s: %s", iface, mac)
					return mac, nil
				}
			}
		}
	}
	
	return "", fmt.Errorf("no active network interface found")
}

// getMACAddressDarwin gets macOS MAC address
func getMACAddressDarwin() (string, error) {
	// Use Go's net package
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	
	for _, iface := range interfaces {
		// Skip loopback and down interfaces
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}
		
		// Get MAC address
		if iface.HardwareAddr != nil && len(iface.HardwareAddr) > 0 {
			mac := iface.HardwareAddr.String()
			if mac != "" && mac != "00:00:00:00:00:00" {
				return mac, nil
			}
		}
	}
	
	return "", fmt.Errorf("no active network interface found")
}