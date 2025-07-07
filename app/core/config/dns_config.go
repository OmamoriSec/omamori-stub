package config

import (
	"errors"
	"omamori/app/core/channels"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

type DNSBackup struct {
	InterfaceName string
	DNSServers    []string
	IsDHCP        bool
}

var (
	isDNSConfigured  = false
	localDNSIP       = "127.0.0.1"
	originalSettings = make(map[string]DNSBackup)
)

func InitDNSConfig(localIP string) {
	localDNSIP = localIP
	originalSettings = make(map[string]DNSBackup)
}

func CheckDNSPrivileges() error {
	switch runtime.GOOS {
	case "linux", "darwin":
		if os.Geteuid() != 0 {
			return errors.New("requires root privileges")
		}
		return nil
	case "windows":
		cmd := exec.Command("netsh", "interface", "show", "interface")
		if err := cmd.Run(); err != nil {
			return errors.New("requires admin privileges")
		}
		return nil
	default:
		return errors.New("unsupported platform: " + runtime.GOOS)
	}
}

func ConfigureSystemDNS() error {
	if isDNSConfigured {
		return errors.New("DNS already configured")
	}

	switch runtime.GOOS {
	case "windows":
		return configureWindowsDNS()
	case "linux":
		return nil
	case "darwin":
		return nil
	default:
		return errors.New("unsupported platform: " + runtime.GOOS)
	}
}

func RestoreSystemDNS() error {
	if !isDNSConfigured {
		return nil
	}

	switch runtime.GOOS {
	case "windows":
		return restoreWindowsDNS()
	case "linux":
		return nil
	case "darwin":
		return nil
	default:
		return errors.New("unsupported platform: " + runtime.GOOS)
	}
}

func configureWindowsDNS() error {
	interfaces, err := getActiveInterfaces()
	if err != nil {
		return err
	}

	if len(interfaces) == 0 {
		return errors.New("no active network interfaces found")
	}

	// Backup current settings (silent)
	for _, iface := range interfaces {
		backupDNSSettings(iface)
	}

	// Configure new DNS settings (silent)
	for _, iface := range interfaces {
		// Set IPv4 DNS (ignore errors)
		runNetshCommand("interface", "ip", "set", "dns", iface, "static", localDNSIP)
		// Set IPv6 DNS (ignore errors)
		runNetshCommand("interface", "ipv6", "set", "dns", iface, "static", "::1")
	}

	isDNSConfigured = true
	logEvent(channels.Log, "🔧 System DNS configured to use Omamori")
	return nil
}

func restoreWindowsDNS() error {
	logEvent(channels.Log, "🔄 Restoring original DNS settings...")

	// Restore silently
	for interfaceName, backup := range originalSettings {
		if backup.IsDHCP {
			runNetshCommand("interface", "ip", "set", "dns", interfaceName, "dhcp")
			runNetshCommand("interface", "ipv6", "set", "dns", interfaceName, "dhcp")
		} else if len(backup.DNSServers) > 0 {
			runNetshCommand("interface", "ip", "set", "dns", interfaceName, "static", backup.DNSServers[0])
			for _, dns := range backup.DNSServers[1:] {
				runNetshCommand("interface", "ip", "add", "dns", interfaceName, dns)
			}
		} else {
			runNetshCommand("interface", "ip", "set", "dns", interfaceName, "dhcp")
		}
	}

	// Reset state
	originalSettings = make(map[string]DNSBackup)
	isDNSConfigured = false
	logEvent(channels.Log, "✅ Original DNS settings restored")
	return nil
}

func getActiveInterfaces() ([]string, error) {
	cmd := exec.Command("netsh", "interface", "show", "interface")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var interfaces []string
	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "Connected") &&
			(strings.Contains(line, "Wi-Fi") || strings.Contains(line, "Ethernet")) {
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				interfaces = append(interfaces, parts[len(parts)-1])
			}
		}
	}
	return interfaces, nil
}

func backupDNSSettings(interfaceName string) error {
	cmd := exec.Command("netsh", "interface", "ip", "show", "dns", interfaceName)
	output, err := cmd.Output()
	if err != nil {
		return err
	}

	backup := DNSBackup{
		InterfaceName: interfaceName,
		DNSServers:    []string{},
		IsDHCP:        false,
	}

	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)

		if strings.Contains(line, "DNS servers configured through DHCP:") {
			backup.IsDHCP = true
			if parts := strings.Split(line, ":"); len(parts) > 1 {
				if dns := strings.TrimSpace(parts[1]); dns != "" && dns != "None" {
					backup.DNSServers = append(backup.DNSServers, dns)
				}
			}
		} else if strings.Contains(line, "Statically Configured DNS Servers:") {
			backup.IsDHCP = false
		} else if strings.Contains(line, ".") && !strings.Contains(line, ":") {
			if dns := strings.TrimSpace(line); dns != "" {
				backup.DNSServers = append(backup.DNSServers, dns)
			}
		}
	}

	originalSettings[interfaceName] = backup
	return nil
}

func runNetshCommand(args ...string) error {
	cmd := exec.Command("netsh", args...)
	_, err := cmd.CombinedOutput()
	return err
}

func logEvent(eventType channels.EventType, message string) {
	channels.LogEventChannel <- channels.Event{
		Type:    eventType,
		Payload: message,
	}
}
