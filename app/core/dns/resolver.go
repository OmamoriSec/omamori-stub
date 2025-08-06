package dns

import (
	"fmt"
	"log"
	"net"
	"omamori/app/core/channels"
	"omamori/app/core/config"
	"omamori/app/core/internal/cache"
	"time"
)

// =============== DNS RELATED METHODS ===============

func resolveCustomDns(domainName string) (string, bool) {
	if customIp := config.BlockedSites.Search(config.ReverseDomain(domainName)); customIp != nil {
		// both 0.0.0.0 and custom Ips will be included in this case
		return *customIp, true
	}
	return "", false
}

func Lookup(dnsQuery *Query) []byte {

	flags := dnsQuery.Header.FLAGS
	// update header according to answer
	dnsQuery.Header.QDCOUNT = 1
	dnsQuery.Header.ARCOUNT = 0

	// Setting QR (bit 15)
	dnsQuery.Header.FLAGS = dnsQuery.Header.FLAGS | 1<<15

	// Setting RA (bit 7)
	dnsQuery.Header.FLAGS = dnsQuery.Header.FLAGS | 1<<7

	encodedName, err := encodeDomainName(dnsQuery.Questions.Name)

	if err != nil {
		log.Printf("Error encoding domain name %s: %s\n", dnsQuery.Questions.Name, err)
		return nil
	}

	if customIP, resolved := resolveCustomDns(dnsQuery.Questions.Name); resolved {
		channels.LogEventChannel <- channels.Event{Type: channels.Log,
			Payload: fmt.Sprintf("Custom DNS lookup enabled for %s: %s\n", dnsQuery.Questions.Name, customIP)}

		dnsQuery.Answer = []*Answer{{
			encodedName,
			dnsQuery.Questions.Type,
			dnsQuery.Questions.Class,
			600,
			1 << 2,
			net.ParseIP(customIP).To4(),
		}}
		dnsQuery.Header.ANCOUNT = 1
		resp, _ := dnsQuery.Encode()
		return resp
	}

	cachedRecord, found := cache.DnsCache.Get(dnsQuery.Questions.Name, dnsQuery.Questions.Type)
	if found {
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
		channels.LogEventChannel <- channels.Event{
			Type:    channels.Log,
			Payload: fmt.Sprintf("Cache hit for %s\n", dnsQuery.Questions.Name),
		}
		resp, _ := dnsQuery.Encode()
		return resp
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

	var upStreamServers = []string{config.Global.Upstream1, config.Global.Upstream2}

	for _, upstream := range upStreamServers {

		conn, err := net.DialUDP("udp4", nil, &net.UDPAddr{IP: net.ParseIP(upstream), Port: 53})

		if err != nil {
			log.Printf("Error %s\n", err)
			continue
		}

		_, err = conn.Write(upstreamQuery)
		if err != nil {
			log.Printf("Error %s\n", err)
			continue
		}

		_ = conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))

		buf := make([]byte, 512)
		n, err := conn.Read(buf)
		if err != nil {
			log.Printf("Error %s\n", err)
			continue
		}

		responseCode := uint16(buf[:n][3] & 0x0F) // get the last 4 bits of the 3rd byte
		// updating RCODE
		dnsQuery.Header.FLAGS = (dnsQuery.Header.FLAGS & 0xFFF0) | responseCode

		fmt.Println(responseCode)

		if responseCode == 2 || responseCode == 5 { // check for the Rcode first, before trying to parse answer
			// case of server failure & refused
			log.Printf("Received error response from upstream: %d", responseCode)
			continue
		} else if responseCode != 0 {
			resp, _ := dnsQuery.Encode()
			dnsQuery.Header.ANCOUNT = 0
			return resp
		}

		responses, err := decodeDnsAnswer(buf[:n])
		if err != nil {
			log.Printf("Error while fetching answer for %s [Record %d] via %s: %s\n", dnsQuery.Questions.Name, dnsQuery.Questions.Type, upstream, err)
			continue
		}

		if len(responses) > 0 {
			// replacing the defaultAnswer with actual responses
			dnsQuery.Answer = responses

			// updating header to with actual number of answers
			dnsQuery.Header.ANCOUNT = uint16(len(responses))

			// caching all the responses
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

	resp, _ := dnsQuery.Encode()
	return resp
}
