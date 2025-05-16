package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"strings"
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

type DNSQuestion struct {
	Name  string
	Type  uint16
	Class uint16
}

type DNSQuery struct {
	Header *DNSHeader
	// As per RFC question section contains a list of questions,
	// Here for simplicity, will only consider 1
	Questions *DNSQuestion
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

func (q *DNSQuestion) Encode() ([]byte, error) {
	buf := new(bytes.Buffer)

	labels := strings.Split(q.Name, ".")

	for _, label := range labels {
		if len(label) > 63 {
			return nil, errors.New("label too long")
		}
		buf.WriteByte(uint8(len(label)))
		buf.WriteString(label)
	}

	buf.WriteByte(0)

	fields := []any{q.Type, q.Class}
	for _, field := range fields {
		if err := binary.Write(buf, binary.BigEndian, field); err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}

func (dq *DNSQuery) Encode() ([]byte, error) {
	buf := new(bytes.Buffer)

	data, err := dq.Header.Encode()
	if err != nil {
		return nil, err
	}
	buf.Write(data)
	data, err = dq.Questions.Encode()
	if err != nil {
		return nil, err
	}
	buf.Write(data)
	return buf.Bytes(), nil
}

func DecodeDNSQuery(data []byte) (*DNSQuery, error) {
	var dq DNSQuery
	header, err := DecodeDNSHeader(data)
	if err != nil {
		return nil, err
	}
	dq.Header = header
	question, err := DecodeDNSQuestion(data, 12)
	if err != nil {
		return nil, err
	}
	dq.Questions = question

	return &dq, nil
}

func DecodeDNSHeader(data []byte) (*DNSHeader, error) {
	if len(data) < 12 {
		return nil, errors.New("malformed DNS header")
	}
	return &DNSHeader{
		ID:      binary.BigEndian.Uint16(data[0:2]),
		FLAGS:   binary.BigEndian.Uint16(data[2:4]),
		QDCOUNT: binary.BigEndian.Uint16(data[4:6]),
		ANCOUNT: binary.BigEndian.Uint16(data[6:8]),
		NSCOUNT: binary.BigEndian.Uint16(data[8:10]),
		ARCOUNT: binary.BigEndian.Uint16(data[10:12]),
	}, nil
}

func DecodeDNSQuestion(data []byte, offset int) (*DNSQuestion, error) {
	var q DNSQuestion
	var labels []string

	for {
		length := int(data[offset])
		if length == 0 {
			offset++
			break
		}

		offset++
		if offset+length > len(data) {
			return nil, errors.New("malformed DNS question")
		}
		labels = append(labels, string(data[offset:offset+length]))
		offset += length
	}
	q.Name = strings.Join(labels, ".")
	if offset+4 > len(data) {
		return &q, errors.New("malformed DNS question")
	}
	q.Type = binary.BigEndian.Uint16(data[offset : offset+2])
	q.Class = binary.BigEndian.Uint16(data[offset+2 : offset+4])
	offset += 4

	return &q, nil
}
