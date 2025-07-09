package config

import (
	"errors"
	"fmt"
	"omamori/app/core/channels"
	"os"
	"os/exec"
	"runtime"
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

// CheckDNSPrivileges verifies if the application has the necessary privileges to modify DNS settings
func CheckDNSPrivileges() error {
	switch runtime.GOOS {
	case "linux":
		if isAndroidEnvironment() {
			return checkAndroidPrivileges()
		}
		if os.Geteuid() != 0 {
			return errors.New("requires root privileges - please run with sudo")
		}
		return nil
	case "darwin":
		cmd := exec.Command("networksetup", "-listallnetworkservices")
		if err := cmd.Run(); err != nil {
			return errors.New("requires admin privileges - please run with sudo")
		}
		return nil
	case "windows":
		err := runCommand("netsh", "interface", "show", "interface")
		if err != nil {
			return errors.New("requires admin privileges - please run as Administrator")
		}
		return nil
	default:
		return errors.New("unsupported platform: " + runtime.GOOS)
	}
}

// ConfigureSystemDNS configures the system to use the local DNS server
func ConfigureSystemDNS() error {
	if isDNSConfigured {
		return errors.New("DNS already configured")
	}

	switch runtime.GOOS {
	case "windows":
		return configureWindowsDNS()
	case "linux":
		if isAndroidEnvironment() {
			return configureAndroidDNS()
		}
		return configureLinuxDNS()
	case "darwin":
		return configureDarwinDNS()
	default:
		return errors.New("unsupported platform: " + runtime.GOOS)
	}
}

// RestoreSystemDNS restores the original DNS settings
func RestoreSystemDNS() error {
	if !isDNSConfigured {
		return nil
	}

	switch runtime.GOOS {
	case "windows":
		return restoreWindowsDNS()
	case "linux":
		if isAndroidEnvironment() {
			return restoreAndroidDNS()
		}
		return restoreLinuxDNS()
	case "darwin":
		return restoreDarwinDNS()
	default:
		return errors.New("unsupported platform: " + runtime.GOOS)
	}
}

// configureWindowsDNS sets up DNS configuration on Windows
func configureWindowsDNS() error {
	interfaces, err := getActiveWindowsInterfaces()
	if err != nil {
		return fmt.Errorf("failed to get active interfaces: %w", err)
	}

	// Backup current settings
	for _, iface := range interfaces {
		if err := backupWindowsDNSSettings(iface); err != nil {
			return fmt.Errorf("failed to backup DNS settings for %s: %w", iface, err)
		}
	}

	// Set DNS for all interfaces
	for _, iface := range interfaces {
		err := runCommand("netsh", "interface", "ip", "set", "dns", iface, "static", localDNSIP)
		if err != nil {
			return fmt.Errorf("failed to set DNS for %s: %w", iface, err)
		}
		err = runCommand("netsh", "interface", "ipv6", "set", "dns", iface, "static", "::1")
		if err != nil {
			return fmt.Errorf("failed to set IPv6 DNS for %s: %w", iface, err)
		}
	}

	isDNSConfigured = true
	logEvent(channels.Log, "🔧 System DNS configured to use Omamori (Windows)")
	return nil
}

// configureLinuxDNS sets up DNS configuration on Linux
func configureLinuxDNS() error {
	// Try systemd-resolved first, fallback to resolv.conf
	if commandExists("resolvectl") {
		return configureSystemdResolved()
	}
	return configureResolvConf()
}

// configureAndroidDNS sets up DNS configuration on Android
func configureAndroidDNS() error {
	if isDeviceRooted() {
		return configureRootedAndroidDNS()
	}

	// Provide manual instructions for non-rooted devices
	logEvent(channels.Log, "⚠️ Non-rooted Android device detected")
	logEvent(channels.Log, "📱 Please configure DNS manually in Wi-Fi settings")
	logEvent(channels.Log, fmt.Sprintf("Set DNS to: %s", localDNSIP))
	return errors.New("manual configuration required for non-rooted devices")
}

// configureDarwinDNS sets up DNS configuration on macOS
func configureDarwinDNS() error {
	services, err := getActiveDarwinServices()
	if err != nil {
		return fmt.Errorf("failed to get network services: %w", err)
	}

	// Backup current settings
	for _, service := range services {
		if err := backupDarwinDNSSettings(service); err != nil {
			return fmt.Errorf("failed to backup DNS settings for %s: %w", service, err)
		}
	}

	// Set DNS for all services
	for _, service := range services {
		if err := setDarwinDNS(service, localDNSIP); err != nil {
			return fmt.Errorf("failed to set DNS for %s: %w", service, err)
		}
	}

	isDNSConfigured = true
	logEvent(channels.Log, "🔧 System DNS configured to use Omamori (macOS)")
	return nil
}

// restoreWindowsDNS restores original DNS settings on Windows
func restoreWindowsDNS() error {
	logEvent(channels.Log, "🔄 Restoring original DNS settings...")

	for interfaceName, backup := range originalSettings {
		if backup.IsDHCP {
			err := runCommand("netsh", "interface", "ip", "set", "dns", interfaceName, "dhcp")
			if err != nil {
				return err
			}
			err = runCommand("netsh", "interface", "ipv6", "set", "dns", interfaceName, "dhcp")
			if err != nil {
				return err
			}
		} else if len(backup.DNSServers) > 0 {
			err := runCommand("netsh", "interface", "ip", "set", "dns", interfaceName, "static", backup.DNSServers[0])
			if err != nil {
				return err
			}
			for _, dns := range backup.DNSServers[1:] {
				err := runCommand("netsh", "interface", "ip", "add", "dns", interfaceName, dns)
				if err != nil {
					return err
				}
			}
		} else {
			err := runCommand("netsh", "interface", "ip", "set", "dns", interfaceName, "dhcp")
			if err != nil {
				return err
			}
		}
	}

	originalSettings = make(map[string]DNSBackup)
	isDNSConfigured = false
	logEvent(channels.Log, "✅ Original DNS settings restored (Windows)")
	return nil
}

// restoreLinuxDNS restores original DNS settings on Linux
func restoreLinuxDNS() error {
	logEvent(channels.Log, "🔄 Restoring original DNS settings...")

	if commandExists("resolvectl") {
		return restoreSystemdResolved()
	}
	return restoreResolvConf()
}

// restoreAndroidDNS restores original DNS settings on Android
func restoreAndroidDNS() error {
	if isDeviceRooted() {
		return restoreRootedAndroidDNS()
	}

	logEvent(channels.Log, "🔄 Please manually restore DNS settings in Wi-Fi configuration")
	return nil
}

// restoreDarwinDNS restores original DNS settings on macOS
func restoreDarwinDNS() error {
	logEvent(channels.Log, "🔄 Restoring original DNS settings...")

	for serviceName, backup := range originalSettings {
		if err := restoreDarwinDNSService(serviceName, backup); err != nil {
			return fmt.Errorf("failed to restore DNS for %s: %w", serviceName, err)
		}
	}

	originalSettings = make(map[string]DNSBackup)
	isDNSConfigured = false
	logEvent(channels.Log, "✅ Original DNS settings restored (macOS)")
	return nil
}
