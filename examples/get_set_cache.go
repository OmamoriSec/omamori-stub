package main

import (
	"fmt"
	"omamori/cache"
	"time"
)

func main() {
	fmt.Println("DNS LRU Cache Example")
	fmt.Println("=====================")

	dnsCache := cache.DNSCache(10)
	defer dnsCache.Close()

	fmt.Println("\n1. Adding entries to the cache")

	ttl := 5 * time.Minute
	googleARecord := &cache.Record{
		Type:      cache.RecordTypeA,
		ExpiresAt: time.Now().Add(ttl),
		Data: &cache.ARecord{
			IPAddress: "142.250.191.78",
		},
	}
	dnsCache.Set("google.com", googleARecord)

	googleAAAARecord := &cache.Record{
		Type:      cache.RecordTypeAAAA,
		ExpiresAt: time.Now().Add(ttl),
		Data: &cache.AAAARecord{
			IPAddress: "2a00:1450:4001:81a::200e",
		},
	}
	dnsCache.Set("google.com", googleAAAARecord)

	exampleCNAMERecord := &cache.Record{
		Type:      cache.RecordTypeCNAME,
		ExpiresAt: time.Now().Add(60 * time.Minute),
		Data: &cache.CNAMERecord{
			Target: "example.com",
		},
	}
	dnsCache.Set("www.example.com", exampleCNAMERecord)

	exampleARecord := &cache.Record{
		Type:      cache.RecordTypeA,
		ExpiresAt: time.Now().Add(60 * time.Minute),
		Data: &cache.ARecord{
			IPAddress: "93.184.216.34",
		},
	}
	dnsCache.Set("example.com", exampleARecord)

	gmailMXRecord := &cache.Record{
		Type:      cache.RecordTypeMX,
		ExpiresAt: time.Now().Add(time.Hour),
		Data: &cache.MXRecord{
			Priority: 10,
			Target:   "alt1.gmail-smtp-in.l.google.com",
		},
	}
	dnsCache.Set("gmail.com", gmailMXRecord)

	githubTXTRecord := &cache.Record{
		Type:      cache.RecordTypeTXT,
		ExpiresAt: time.Now().Add(24 * time.Minute),
		Data: &cache.TXTRecord{
			Text: []string{"v=spf1 ip4:192.30.252.0/22 include:_netblocks.google.com ~all"},
		},
	}
	dnsCache.Set("github.com", githubTXTRecord)

	nxRecord := &cache.Record{
		Type:      cache.RecordTypeNXDomain,
		ExpiresAt: time.Now().Add(10 * time.Minute),
		Data:      &cache.NXDomainRecord{},
	}
	dnsCache.Set("non-existent-domain.example", nxRecord)

	fmt.Println("\n2. Looking up entries in the cache")
	lookup(dnsCache, "google.com", cache.RecordTypeA)
	lookup(dnsCache, "google.com", cache.RecordTypeAAAA)
	lookup(dnsCache, "www.example.com", cache.RecordTypeCNAME)

	// non-existing domain
	lookup(dnsCache, "non-existent-domain.example", cache.RecordTypeNXDomain)
	lookup(dnsCache, "does-not-exists.example", cache.RecordTypeA)

	fmt.Println("\nDone!")
}

func lookup(dnsCache *cache.LRUCache, domain string, recordType cache.RecordType) {
	record, found := dnsCache.Get(domain, recordType)
	if !found {
		fmt.Printf("❌ Domain %s not found in cache\n", domain)
		return
	}
	fmt.Printf("✅ Found %s in cache: %s (expires in %.f seconds)\n",
		recordType, domain, time.Until(record.ExpiresAt).Seconds())

	switch recordType {
	case cache.RecordTypeA:
		data := record.Data.(*cache.ARecord)
		fmt.Printf("    IP Address: %s\n\n", data.IPAddress)
	case cache.RecordTypeAAAA:
		data := record.Data.(*cache.AAAARecord)
		fmt.Printf("    IP Address: %s\n\n", data.IPAddress)
	case cache.RecordTypeCNAME:
		data := record.Data.(*cache.CNAMERecord)
		fmt.Printf("    Target: %s\n\n", data.Target)
	case cache.RecordTypeMX:
		data := record.Data.(*cache.MXRecord)
		fmt.Printf("    Priority: %d\n    Target: %s\n\n", data.Priority, data.Target)
	case cache.RecordTypeTXT:
		data := record.Data.(*cache.TXTRecord)
		fmt.Printf("    Text: %v\n\n", data.Text)
	case cache.RecordTypeNXDomain:
		fmt.Printf("    Domain does not exist (negative cache entry)\n")
	}
}
