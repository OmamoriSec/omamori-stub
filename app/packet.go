package main

import (
	"bytes"
	"encoding/binary"
)

type DNSHeader struct {
	ID uint16 // Packet Identifier
	// Flags is a 16-bit field that includes QR, OPCODE, AA, TC, RD, RA, Z, and RCODE
	/*

		| QR | Opcode (4 bits) | AA | TC | RD | RA | Z (3 bits) | RCODE (4 bits) |
		|----|------------------|----|----|----|----|------------|----------------|
		 15   14-11             10   9    8    7    6-4          3-0

	*/
	FLAGS   uint16
	QDCOUNT uint16 // Question Count (QDCOUNT)
	ANCOUNT uint16 // Answer Record Count (ANCOUNT)
	NSCOUNT uint16 // Authority Record Count (NSCOUNT)
	ARCOUNT uint16 // Additional Record Count (ARCOUNT)
}

func (h *DNSHeader) Encode() ([]byte, error) {
	buf := new(bytes.Buffer)
	fields := []any{h.ID, h.FLAGS, h.QDCOUNT, h.ANCOUNT, h.NSCOUNT, h.ARCOUNT}

	for _, field := range fields {
		if err := binary.Write(buf, binary.BigEndian, field); err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}
