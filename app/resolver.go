package main

import (
	"fmt"
	"net"
)

// =============== DNS RELATED METHODS ===============

func resolveable(domain string) bool {
	if blocked := blockedSites.search(reverseDomain(domain)); blocked {
		return false
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
