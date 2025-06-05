package main

import (
	"fmt"
	"net"
	cache2 "omamori/app/internal/cache"
	"time"
)

func main() {
	fmt.Println("DNS LRU Cache Example")
	fmt.Println("=====================")

	dnsCache := cache2.DNSCache(10)
	defer dnsCache.Close()

	fmt.Println("\n1. Adding entries to the cache")

	ttl := 5 * time.Minute
	response := "142.250.191.78"
	googleARecord := &cache2.Record{
		Type:      cache2.RecordType(1),
		ExpiresAt: time.Now().Add(ttl),
		Data:      net.ParseIP(response).To4(),
	}
	dnsCache.Set("google.com", googleARecord)

	response = "2a00:1450:4001:81a::200e"
	googleAAAARecord := &cache2.Record{
		Type:      cache2.RecordType(28),
		ExpiresAt: time.Now().Add(ttl),
		Data:      net.ParseIP(response).To4(),
	}
	dnsCache.Set("google.com", googleAAAARecord)

	response = "example.com"
	exampleCNAMERecord := &cache2.Record{
		Type:      cache2.RecordType(5),
		ExpiresAt: time.Now().Add(60 * time.Minute),
		Data:      net.ParseIP(response).To4(),
	}
	dnsCache.Set("www.example.com", exampleCNAMERecord)

	response = "93.184.216.34"
	exampleARecord := &cache2.Record{
		Type:      cache2.RecordType(1),
		ExpiresAt: time.Now().Add(60 * time.Minute),
		Data:      net.ParseIP(response).To4(),
	}
	dnsCache.Set("example.com", exampleARecord)

	response = "alt1.gmail-smtp-in.l.google.com"
	gmailMXRecord := &cache2.Record{
		Type:      cache2.RecordType(15),
		ExpiresAt: time.Now().Add(time.Hour),
		Data:      net.ParseIP(response).To4(),
	}
	dnsCache.Set("gmail.com", gmailMXRecord)

	response = "v=spf1 ip4:192.30.252.0/22 include:_netblocks.google.com ~all"
	githubTXTRecord := &cache2.Record{
		Type:      cache2.RecordType(16),
		ExpiresAt: time.Now().Add(24 * time.Minute),
		Data:      net.ParseIP(response).To4(),
	}
	dnsCache.Set("github.com", githubTXTRecord)

	response = ""
	nxRecord := &cache2.Record{
		Type:      cache2.RecordType(65535),
		ExpiresAt: time.Now().Add(10 * time.Minute),
		Data:      net.ParseIP(response).To4(),
	}
	dnsCache.Set("non-existent-domain.example", nxRecord)

	fmt.Println("\n2. Looking up entries in the cache")
	lookup(dnsCache, "google.com", 1)
	lookup(dnsCache, "google.com", 28)
	lookup(dnsCache, "www.example.com", 5)

	// non-existing domain
	lookup(dnsCache, "non-existent-domain.example", 65535)
	lookup(dnsCache, "does-not-exists.example", 1)

	fmt.Println("\n3. Printing cache contents")
	dnsCache.PrintCacheContents()

	fmt.Println("\nDone!")
}

func lookup(dnsCache *cache2.LRUCache, domain string, recordType uint16) {
	record, found := dnsCache.Get(domain, recordType)
	if !found {
		fmt.Printf("❌ Domain %s not found in cache\n", domain)
		return
	}
	fmt.Printf("✅ Found %d in cache: %s (expires in %.f seconds)\n",
		recordType, domain, time.Until(record.ExpiresAt).Seconds())
}
