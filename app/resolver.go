package main

import (
	"fmt"
	"net"
	"os"
	"strings"
)

var blockedSites = make(map[string]struct{})

func loadBlockedSites(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	lines := strings.Split(string(data), "\n")

	for _, line := range lines {
		domain := strings.TrimSpace(line)
		if domain != "" {
			blockedSites[domain] = struct{}{}
		}
	}

	return nil
}

func resolveable(domain string) bool {
	if _, blocked := blockedSites[domain]; blocked {
		return false
	}

	// check for subdomain
	parts := strings.Split(domain, ".")
	for i := 1; i < len(parts); i++ {
		sub := strings.Join(parts[i:], ".")
		if _, blocked := blockedSites[sub]; blocked {
			return false
		}

	}
	return true
}

func lookup(dnsQuery *DNSQuery) *DNSQuery {

	// update header according to answer
	dnsQuery.Header.QDCOUNT = 1
	dnsQuery.Header.ARCOUNT = 0
	dnsQuery.Header.ANCOUNT = 1

	// Setting QR (bit 15)
	dnsQuery.Header.FLAGS = dnsQuery.Header.FLAGS | 1<<15

	// Setting RA (bit 7)
	dnsQuery.Header.FLAGS = dnsQuery.Header.FLAGS | 1<<7

	if !resolveable(dnsQuery.Questions.Name) {
		fmt.Printf("%s is not resolvable\n", dnsQuery.Questions.Name)

		// create answer
		dnsQuery.Answer = &DNSAnswer{dnsQuery.Questions.Name,
			dnsQuery.Questions.Type,
			dnsQuery.Questions.Class,
			60,
			1 << 2,
			net.ParseIP("0.0.0.0").To4(),
		}

		return dnsQuery
	}

	// TODO: @CosmicOppai make query to the upstream DNS resolver
	// update answer as per upstream
	dnsQuery.Answer = &DNSAnswer{dnsQuery.Questions.Name,
		dnsQuery.Questions.Type,
		dnsQuery.Questions.Class,
		60,
		1 << 2,
		net.ParseIP("8.8.8.8").To4(),
	}

	return dnsQuery

}
