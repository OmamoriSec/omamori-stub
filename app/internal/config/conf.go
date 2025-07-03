package config

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"omamori/app/internal/radix"
	"os"
	"path/filepath"
	"strings"
)

// =============== CONFIGURATIONS ===============

var BlockedSites *radix.Tree

type Config struct {
	Upstream2     string `json:"upstream2"`
	Upstream1     string `json:"upstream1"`
	CertPath      string `json:"cert_path"`
	KeyPath       string `json:"key_path"`
	UdpServerPort int    `json:"port"`
	MapFile       string `json:"map_file"`
	ConfigFile    string `json:"-"`
	ConfigDir     string `json:"-"`
}

var Global = NewConfig()

func LoadBlockedSites() error {
	BlockedSites = radix.NewRadixTree()

	blockedFilePath := Global.MapFile

	_, err := os.Stat(blockedFilePath)
	// Download File if it doesn't exist

	if err != nil {
		resp, err := http.Get("https://raw.githubusercontent.com/StevenBlack/hosts/master/hosts")
		if err != nil || resp.StatusCode != 200 {
			return err
		}
		defer func(Body io.ReadCloser) {
			_ = Body.Close()
		}(resp.Body)

		outFile, err := os.Create(blockedFilePath)

		if err != nil {
			log.Println(err)
			return err
		}

		defer func(outFile *os.File) {
			_ = outFile.Close()
		}(outFile)

		_, err = io.Copy(outFile, resp.Body)

		if err != nil {
			return err
		}
	}

	data, err := os.ReadFile(blockedFilePath)
	if err != nil {
		return err
	}

	lines := strings.Split(string(data), "\n")

	for _, line := range lines {
		entry := strings.TrimSpace(line)

		if entry == "" || strings.HasPrefix(entry, "#") {
			continue
		}

		var domain string

		spaceIndex := strings.Index(entry, " ")
		if spaceIndex != -1 {
			endIndex := strings.Index(entry[spaceIndex+1:], " ")
			if endIndex == -1 {
				// there is no extra comment or space after the domain name
				domain = strings.TrimSpace(line[spaceIndex+1:])
			} else {
				domain = strings.TrimSpace(entry[spaceIndex+1 : endIndex])
			}
			BlockedSites.Insert(ReverseDomain(domain), strings.TrimSpace(entry[:spaceIndex]))
		}
	}

	siteList := BlockedSites.GetItems()
	for site, ip := range siteList {
		log.Printf("Site: %s, IP: %s", ReverseDomain(site), ip)
	}
	os.Exit(0)

	return nil
}

func ReverseDomain(domain string) string {
	parts := strings.Split(domain, ".")
	for i, j := 0, len(parts)-1; i < j; i, j = i+1, j-1 {
		parts[i], parts[j] = parts[j], parts[i]
	}
	return strings.Join(parts, ".")
}

func isValidIP(ip string) bool {
	return net.ParseIP(ip) != nil
}

func LoadConfig() error {
	data, err := os.ReadFile(Global.ConfigFile)
	if err != nil {
		return err
	}

	parsedConfig := Config{}
	err = json.Unmarshal(data, &parsedConfig)
	if err != nil {
		return err
	}

	if isValidIP(parsedConfig.Upstream2) && isValidIP(parsedConfig.Upstream1) {
		Global.Upstream2 = parsedConfig.Upstream2
		Global.Upstream1 = parsedConfig.Upstream1
	}

	if parsedConfig.UdpServerPort > 0 && parsedConfig.UdpServerPort < 65535 {
		Global.UdpServerPort = parsedConfig.UdpServerPort
	}

	if _, err = os.Stat(parsedConfig.MapFile); err == nil {
		Global.MapFile = parsedConfig.MapFile
	}

	if _, err = os.Stat(parsedConfig.KeyPath); err == nil {
		Global.KeyPath = parsedConfig.KeyPath
	}

	if _, err = os.Stat(parsedConfig.CertPath); err == nil {
		Global.CertPath = parsedConfig.CertPath
	}

	return nil
}

func EnsureDefaultConfig() (*Config, error) {
	// 0700 for directory: rwx------ - only owner (root, since it's in /etc/) can read/write/execute
	// 0600 for file: rw------- - only owner can read/write

	if _, err := os.Stat(Global.ConfigFile); err == nil {
		// if config file exits, parse it
		_ = LoadConfig()
		return Global, nil
	}

	// Create dir if it doesn't exist
	if err := os.MkdirAll(Global.ConfigDir, 0700); err != nil {
		return nil, err
	}

	// Write default config
	defaultConfig := fmt.Sprintf(`{
	    "upstream1": "%s",
	    "upstream2": "%s",
        "cert_path": "%s",
        "key_path": "%s",
        "port": %d,
		"map_file": "%s"
    }`, Global.Upstream1, Global.Upstream2, Global.CertPath, Global.KeyPath, Global.UdpServerPort, Global.MapFile)

	return Global, os.WriteFile(Global.ConfigFile, []byte(defaultConfig), 0600)
}

func NewConfig() *Config {
	configDir := "/etc/omamori"
	configFile := filepath.Join(configDir, "config.json")
	mapFile := filepath.Join(configDir, "map.txt")

	const (
		upstream1 = "1.1.1.1"
		upstream2 = "208.67.220.220"
		certPath  = "/etc/omamori/cert/server.crt"
		keyPath   = "/etc/omamori/cert/server.key"
		port      = 2053
	)

	return &Config{
		MapFile:       mapFile,
		Upstream1:     upstream1,
		Upstream2:     upstream2,
		CertPath:      certPath,
		KeyPath:       keyPath,
		UdpServerPort: port,
		ConfigFile:    configFile,
		ConfigDir:     configDir,
	}
}
