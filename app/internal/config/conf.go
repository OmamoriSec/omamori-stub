package config

import (
	"errors"
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

var BlockedSites *radix.RadixTree
var (
	Upstream2 string
	Upstream1 string
)

func LoadBlockedSites(filename string) error {
	BlockedSites = radix.NewRadixTree()

	blockedFilePath := filepath.Join(filepath.Dir(filename), filename)

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

		outFile, err := os.Create(filename)

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

	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	lines := strings.Split(string(data), "\n")

	for _, line := range lines {
		domain := strings.TrimSpace(line)

		if domain == "" || strings.HasPrefix(domain, "#") {
			continue
		}

		spaceIndex := strings.Index(domain, " ")
		if spaceIndex != -1 {
			endIndex := strings.Index(domain[spaceIndex+1:], " ")
			if endIndex == -1 {
				// there is no extra comment or space after the domain name
				domain = strings.TrimSpace(line[spaceIndex+1:])
			} else {
				domain = strings.TrimSpace(domain[spaceIndex+1 : endIndex])
			}
			BlockedSites.Insert(ReverseDomain(domain))
		}
	}

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

func LoadUpstreamConf(filename string) error {

	blockedFilePath := filepath.Join(filepath.Dir(filename), filename)

	_, err := os.Stat(blockedFilePath)
	// Download File if it doesn't exist

	if err != nil {
		Upstream1 = "1.1.1.1"        // cloudflare
		Upstream2 = "208.67.220.220" // open dns
		return nil
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	lines := strings.Split(string(data), "\n")

	confMap := make(map[string]string)

	for _, line := range lines {
		conf := strings.TrimSpace(line)
		if conf == "" || strings.HasPrefix(conf, "#") {
			continue // skip empty lines and comments
		}

		spaceIndex := strings.Index(conf, " ")
		if spaceIndex != -1 {
			confMap[conf[:spaceIndex]] = conf[spaceIndex+1:]
		}
	}

	u1, u2 := confMap["UPSTREAM1"], confMap["UPSTREAM2"]

	if !isValidIP(u1) || !isValidIP(u2) {
		return errors.New("invalid IP address in config file")
	}

	Upstream1 = u1
	Upstream2 = u2

	return nil
}
