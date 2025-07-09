package config

import (
	"errors"
	"fmt"
	"omamori/app/core/channels"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

// Helper functions - Windows
func getActiveWindowsInterfaces() ([]string, error) {
	cmd := exec.Command("netsh", "interface", "show", "interface")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
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

func backupWindowsDNSSettings(interfaceName string) error {
	cmd := exec.Command("netsh", "interface", "ip", "show", "dns", interfaceName)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
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
		} else if strings.Contains(line, ".") && !strings.Contains(line, ":") {
			if dns := strings.TrimSpace(line); dns != "" {
				backup.DNSServers = append(backup.DNSServers, dns)
			}
		}
	}

	originalSettings[interfaceName] = backup
	return nil
}

// Helper functions - Linux
func configureSystemdResolved() error {
	// Backup current configuration
	if err := backupFile("/etc/systemd/resolved.conf", "/etc/systemd/resolved.conf.omamori.bak"); err != nil {
		return err
	}

	config := fmt.Sprintf(`[Resolve]
DNS=%s
`, localDNSIP)

	if err := os.WriteFile("/etc/systemd/resolved.conf", []byte(config), 0644); err != nil {
		return err
	}

	if err := runCommand("systemctl", "restart", "systemd-resolved"); err != nil {
		return fmt.Errorf("failed to restart systemd-resolved: %w", err)
	}

	isDNSConfigured = true
	logEvent(channels.Log, "🔧 System DNS configured to use Omamori (systemd-resolved)")
	return nil
}

func configureResolvConf() error {
	if err := backupFile("/etc/resolv.conf", "/etc/resolv.conf.omamori.bak"); err != nil {
		return err
	}

	config := fmt.Sprintf("nameserver %s\n", localDNSIP)
	if err := os.WriteFile("/etc/resolv.conf", []byte(config), 0644); err != nil {
		return err
	}

	isDNSConfigured = true
	logEvent(channels.Log, "🔧 System DNS configured to use Omamori (resolv.conf)")
	return nil
}

func restoreSystemdResolved() error {
	// Clear per-interface settings first
	interfaces, _ := getActiveLinuxInterfaces()
	for _, iface := range interfaces {
		runCommand("resolvectl", "dns", iface, "")
		runCommand("resolvectl", "domain", iface, "")
		runCommand("resolvectl", "default-route", iface, "no")
	}

	// Restore original resolved.conf
	if err := restoreFile("/etc/systemd/resolved.conf.omamori.bak", "/etc/systemd/resolved.conf", "systemctl restart systemd-resolved"); err != nil {
		return err
	}

	isDNSConfigured = false
	logEvent(channels.Log, "✅ Original DNS settings restored (systemd-resolved)")
	return nil
}

func restoreResolvConf() error {
	if err := restoreFile("/etc/resolv.conf.omamori.bak", "/etc/resolv.conf", ""); err != nil {
		return err
	}

	isDNSConfigured = false
	logEvent(channels.Log, "✅ Original DNS settings restored (resolv.conf)")
	return nil
}

// Helper functions - Android
func configureRootedAndroidDNS() error {
	logEvent(channels.Log, "🔧 Configuring DNS on rooted Android device...")

	// Backup current settings
	if err := backupAndroidDNSSettings(); err != nil {
		return fmt.Errorf("failed to backup DNS settings: %w", err)
	}

	// Get active network interface
	iface, err := getActiveAndroidInterface()
	if err != nil {
		return fmt.Errorf("failed to get active interface: %w", err)
	}

	// Method 1: Using setprop (property modification)
	if err := setAndroidProperty("net.dns1", localDNSIP); err == nil {
		if err := setAndroidProperty("net.dns2", localDNSIP); err == nil {
			// Restart network to apply changes
			if err := restartAndroidNetwork(iface); err != nil {
				logEvent(channels.Log, "⚠️ Network restart failed, DNS may not be fully applied")
			}
			isDNSConfigured = true
			logEvent(channels.Log, "🔧 System DNS configured to use Omamori (setprop)")
			return nil
		}
	}

	// Method 2: Direct iptables DNS redirection (if setprop fails)
	if err := configureAndroidIPTables(); err != nil {
		return fmt.Errorf("failed to configure iptables DNS redirection: %w", err)
	}

	isDNSConfigured = true
	logEvent(channels.Log, "🔧 System DNS configured to use Omamori (iptables)")
	return nil
}

func restoreRootedAndroidDNS() error {
	logEvent(channels.Log, "🔄 Restoring original DNS settings...")

	// Restore properties
	if backup, exists := originalSettings["android_dns1"]; exists {
		if len(backup.DNSServers) > 0 {
			setAndroidProperty("net.dns1", backup.DNSServers[0])
		}
	}
	if backup, exists := originalSettings["android_dns2"]; exists {
		if len(backup.DNSServers) > 0 {
			setAndroidProperty("net.dns2", backup.DNSServers[0])
		}
	}

	// Clear iptables rules
	clearAndroidIPTables()

	isDNSConfigured = false
	logEvent(channels.Log, "✅ Original DNS settings restored (Android)")
	return nil
}

// Helper functions - macOS
func getActiveDarwinServices() ([]string, error) {
	cmd := exec.Command("networksetup", "-listallnetworkservices")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var services []string
	lines := strings.Split(string(output), "\n")

	for i := 1; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" || strings.HasPrefix(line, "*") {
			continue
		}
		if isServiceActive(line) {
			services = append(services, line)
		}
	}

	return services, nil
}

func backupDarwinDNSSettings(serviceName string) error {
	cmd := exec.Command("networksetup", "-getdnsservers", serviceName)
	output, err := cmd.Output()
	if err != nil {
		return err
	}

	backup := DNSBackup{
		InterfaceName: serviceName,
		DNSServers:    []string{},
		IsDHCP:        false,
	}

	outputStr := strings.TrimSpace(string(output))
	if strings.Contains(outputStr, "There aren't any DNS Servers set") {
		backup.IsDHCP = true
	} else {
		lines := strings.Split(outputStr, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" && !strings.Contains(line, "There aren't any DNS Servers set") {
				backup.DNSServers = append(backup.DNSServers, line)
			}
		}
	}

	originalSettings[serviceName] = backup
	return nil
}

func setDarwinDNS(serviceName, dnsIP string) error {
	cmd := exec.Command("networksetup", "-setdnsservers", serviceName, dnsIP)
	return cmd.Run()
}

func restoreDarwinDNSService(serviceName string, backup DNSBackup) error {
	if len(backup.DNSServers) > 0 {
		args := []string{"-setdnsservers", serviceName}
		args = append(args, backup.DNSServers...)
		cmd := exec.Command("networksetup", args...)
		return cmd.Run()
	} else {
		cmd := exec.Command("networksetup", "-setdnsservers", serviceName, "empty")
		return cmd.Run()
	}
}

// Utility functions
func isAndroidEnvironment() bool {
	// Check for Android-specific paths and properties
	androidIndicators := []string{
		"/system/build.prop",
		"/data/data",
		"/android_root",
	}

	for _, path := range androidIndicators {
		if _, err := os.Stat(path); err == nil {
			return true
		}
	}

	// Check if getprop command exists (Android-specific)
	if _, err := exec.LookPath("getprop"); err == nil {
		return true
	}

	return false
}

func isDeviceRooted() bool {
	// Check common indicators of root access
	rootIndicators := []string{
		"/system/bin/su",
		"/system/xbin/su",
		"/sbin/su",
		"/system/su",
		"/vendor/bin/su",
	}

	for _, path := range rootIndicators {
		if _, err := os.Stat(path); err == nil {
			return true
		}
	}

	// Try to execute 'su' command
	cmd := exec.Command("su", "-c", "id")
	if err := cmd.Run(); err == nil {
		return true
	}

	return false
}

func checkAndroidPrivileges() error {
	if !isDeviceRooted() {
		return errors.New("requires root access or manual configuration")
	}
	cmd := exec.Command("su", "-c", "id")
	if err := cmd.Run(); err != nil {
		return errors.New("root access denied - please grant superuser permissions")
	}
	return nil
}

func setAndroidProperty(prop, value string) error {
	cmd := exec.Command("su", "-c", fmt.Sprintf("setprop %s %s", prop, value))
	_, err := cmd.CombinedOutput()
	return err
}

func getAndroidProperty(prop string) (string, error) {
	cmd := exec.Command("su", "-c", fmt.Sprintf("getprop %s", prop))
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func backupAndroidDNSSettings() error {
	dns1, err := getAndroidProperty("net.dns1")
	if err == nil && dns1 != "" {
		originalSettings["android_dns1"] = DNSBackup{
			InterfaceName: "android_dns1",
			DNSServers:    []string{dns1},
			IsDHCP:        false,
		}
	}

	dns2, err := getAndroidProperty("net.dns2")
	if err == nil && dns2 != "" {
		originalSettings["android_dns2"] = DNSBackup{
			InterfaceName: "android_dns2",
			DNSServers:    []string{dns2},
			IsDHCP:        false,
		}
	}

	return nil
}

func isServiceActive(serviceName string) bool {
	cmd := exec.Command("networksetup", "-getinfo", serviceName)
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	outputStr := string(output)
	return strings.Contains(outputStr, "IP address:") ||
		strings.Contains(outputStr, "DHCP Configuration") ||
		strings.Contains(outputStr, "Manual Configuration")
}

func commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

func backupFile(src, dest string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dest, data, 0644)
}

func restoreFile(src, dest, restartCmd string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	if err := os.WriteFile(dest, data, 0644); err != nil {
		return err
	}

	os.Remove(src)

	if restartCmd != "" {
		parts := strings.Fields(restartCmd)
		cmd := exec.Command(parts[0], parts[1:]...)
		cmd.Run()
	}
	return nil
}

func runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	_, err := cmd.CombinedOutput()
	return err
}

func logEvent(eventType channels.EventType, message string) {
	channels.LogEventChannel <- channels.Event{
		Type:    eventType,
		Payload: message,
	}
}

// Additional Android helper functions
func getActiveAndroidInterface() (string, error) {
	cmd := exec.Command("su", "-c", "ip route show default")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	for _, line := range strings.Split(string(output), "\n") {
		if strings.Contains(line, "dev ") {
			parts := strings.Fields(line)
			for i, part := range parts {
				if part == "dev" && i+1 < len(parts) {
					return parts[i+1], nil
				}
			}
		}
	}

	return "", errors.New("no active interface found")
}

func restartAndroidNetwork(iface string) error {
	cmd := exec.Command("su", "-c", fmt.Sprintf("ip link set %s down && ip link set %s up", iface, iface))
	_, err := cmd.CombinedOutput()
	return err
}

func configureAndroidIPTables() error {
	rules := []string{
		fmt.Sprintf("iptables -t nat -A OUTPUT -p udp --dport 53 -j DNAT --to-destination %s:%d",
			localDNSIP, Global.UdpServerPort),
		fmt.Sprintf("iptables -t nat -A OUTPUT -p tcp --dport 53 -j DNAT --to-destination %s:%d",
			localDNSIP, Global.UdpServerPort),
	}

	for _, rule := range rules {
		cmd := exec.Command("su", "-c", rule)
		if _, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to apply iptables rule: %s, error: %w", rule, err)
		}
	}

	return nil
}

func clearAndroidIPTables() error {
	rules := []string{
		fmt.Sprintf("iptables -t nat -D OUTPUT -p udp --dport 53 -j DNAT --to-destination %s:%d",
			localDNSIP, Global.UdpServerPort),
		fmt.Sprintf("iptables -t nat -D OUTPUT -p tcp --dport 53 -j DNAT --to-destination %s:%d",
			localDNSIP, Global.UdpServerPort),
	}

	for _, rule := range rules {
		cmd := exec.Command("su", "-c", rule)
		cmd.CombinedOutput() // Ignore errors when clearing
	}

	return nil
}

func getActiveLinuxInterfaces() ([]string, error) {
	cmd := exec.Command("ip", "-o", "link", "show", "up")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var interfaces []string
	for _, line := range strings.Split(string(output), "\n") {
		if strings.Contains(line, ": ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				iface := strings.TrimSuffix(parts[1], ":")
				if iface != "lo" { // Skip loopback
					interfaces = append(interfaces, iface)
				}
			}
		}
	}
	return interfaces, nil
}
