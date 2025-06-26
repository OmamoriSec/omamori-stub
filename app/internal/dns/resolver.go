package dns

import (
	"log"
	"net"
	"omamori/app/internal/cache"
	"omamori/app/internal/config"
	"time"
)

// =============== DNS RELATED METHODS ===============

func resolveable(domain string) bool {
	if blocked := config.BlockedSites.Search(config.ReverseDomain(domain)); blocked {
		return false
	}
	return true
}

func Lookup(dnsQuery *Query) *Query {

	flags := dnsQuery.Header.FLAGS
	// update header according to answer
	dnsQuery.Header.QDCOUNT = 1
	dnsQuery.Header.ARCOUNT = 0
	dnsQuery.Header.ANCOUNT = 1

	// Setting QR (bit 15)
	dnsQuery.Header.FLAGS = dnsQuery.Header.FLAGS | 1<<15

	// Setting RA (bit 7)
	dnsQuery.Header.FLAGS = dnsQuery.Header.FLAGS | 1<<7

	encodedName, err := encodeDomainName(dnsQuery.Questions.Name)
	if err != nil {
		log.Printf("Error encoding domain name %s: %s\n", dnsQuery.Questions.Name, err)
		return dnsQuery
	}

	defaultAnswer := &Answer{
		encodedName,
		dnsQuery.Questions.Type,
		dnsQuery.Questions.Class,
		600,
		1 << 2,
		net.ParseIP("0.0.0.0").To4(),
	}
	dnsQuery.Answer = []*Answer{defaultAnswer}

	if !resolveable(dnsQuery.Questions.Name) {
		log.Printf("%s is not resolvable\n", dnsQuery.Questions.Name)
		return dnsQuery
	}

	cachedRecord, found := cache.DnsCache.Get(dnsQuery.Questions.Name, dnsQuery.Questions.Type)
	if found {
		encodedName, err := encodeDomainName(dnsQuery.Questions.Name)
		if err != nil {
			log.Printf("Error encoding domain name %s: %s\n", dnsQuery.Questions.Name, err)
			return dnsQuery
		}

		// update answer
		cachedAnswer := &Answer{
			encodedName,
			dnsQuery.Questions.Type,
			dnsQuery.Questions.Class,
			uint32(time.Until(cachedRecord.ExpiresAt).Seconds()),
			uint16(len(cachedRecord.Data)),
			cachedRecord.Data,
		}

		dnsQuery.Answer = []*Answer{cachedAnswer}
		dnsQuery.Header.ANCOUNT = 1

		log.Printf("Cache hit for %s\n", dnsQuery.Questions.Name)
		return dnsQuery
	}

	// update answer as per upstream

	upstreamQuery, _ := (&Query{
		Header: &Header{
			ID:      dnsQuery.Header.ID,
			FLAGS:   flags,
			QDCOUNT: 1,
			ANCOUNT: 0,
			NSCOUNT: 0,
			ARCOUNT: 0,
		},
		Questions: &Question{
			Name:  dnsQuery.Questions.Name,
			Type:  dnsQuery.Questions.Type,
			Class: dnsQuery.Questions.Class,
		},
	}).Encode()

	var upStreamServers = []string{config.Upstream1, config.Upstream2}

	for _, upstream := range upStreamServers {

		conn, err := net.DialUDP("udp4", nil, &net.UDPAddr{IP: net.ParseIP(upstream), Port: 53})

		if err != nil {
			log.Printf("Error %s\n", err)
			return dnsQuery
		}

		_, err = conn.Write(upstreamQuery)
		if err != nil {
			log.Printf("Error %s\n", err)
			continue
		}

		_ = conn.SetReadDeadline(time.Now().Add(1 * time.Second))

		buf := make([]byte, 512)
		n, err := conn.Read(buf)
		if err != nil {
			log.Printf("Error %s\n", err)
			continue
		}

		responses, err := decodeDnsAnswer(buf[:n], dnsQuery)
		if err != nil {
			log.Printf("Error while fetching answer for %s via %s: %s\n", dnsQuery.Questions.Name, upstream, err)
			continue
		}

		if len(responses) > 0 {
			// replacing the defaultAnswer with actual responses
			dnsQuery.Answer = responses

			// updating header to with actual number of answers
			dnsQuery.Header.ANCOUNT = uint16(len(responses))

			// cacnhing all the responses
			for _, response := range responses {
				go cache.DnsCache.Set(dnsQuery.Questions.Name, &cache.Record{
					Type:      cache.RecordType(response.Type),
					ExpiresAt: time.Now().Add(time.Duration(response.TTL) * time.Second),
					Data:      response.Data,
				})
			}
		}

		break
	}

	return dnsQuery
}
