package main

import (
	"fmt"
	"net"
	"omamori/app/cache"
	"time"
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
		data := net.ParseIP("0.0.0.0").To4()
		dnsQuery.Answer = &DNSAnswer{dnsQuery.Questions.Name,
			dnsQuery.Questions.Type,
			dnsQuery.Questions.Class,
			60,
			1 << 2,
			data,
		}

		return dnsQuery
	}

	cachedRecord, found := cache.DnsCache.Get(dnsQuery.Questions.Name, dnsQuery.Questions.Type)
	if found {
		// update answer
		dnsQuery.Answer = &DNSAnswer{dnsQuery.Questions.Name,
			dnsQuery.Questions.Type,
			dnsQuery.Questions.Class,
			60,
			1 << 2,
			cachedRecord.Data,
		}
		fmt.Printf("Cache hit for %s\n", dnsQuery.Questions.Name)

		return dnsQuery
	}

	// TODO: @CosmicOppai make query to the upstream DNS resolver
	// update answer as per upstream
	responseIP := "8.8.8.8"
	data := net.ParseIP(responseIP).To4()
	cache.DnsCache.Set(dnsQuery.Questions.Name, &cache.Record{
		Type:      cache.RecordType(dnsQuery.Questions.Type),
		ExpiresAt: time.Now().Add(5 * time.Minute),
		Data:      data,
	})

	dnsQuery.Answer = &DNSAnswer{dnsQuery.Questions.Name,
		dnsQuery.Questions.Type,
		dnsQuery.Questions.Class,
		60,
		1 << 2,
		data,
	}

	return dnsQuery
}
