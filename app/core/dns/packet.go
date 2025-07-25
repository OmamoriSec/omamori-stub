package dns

import (
	"bytes"
	"encoding/binary"
	"errors"
	"strings"
)

// -- STRUCT START -- //

type Header struct {
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

type Question struct {
	Name  string // TODO! convert this to []byte for consistency
	Type  uint16
	Class uint16
}

type Answer struct {
	Name   []byte // changed to []byte to preserve original name bytes
	Type   uint16
	Class  uint16
	TTL    uint32
	Length uint16
	Data   []byte
}

type Query struct {
	Header *Header
	// As per RFC question section contains a list of questions,
	// Here for simplicity, will only consider 1
	Questions *Question
	Answer    []*Answer

	// TODO: for future
	//Authority []*SOA
	// AdditionalRecord []*ARN
}

// -- STRUCT END -- //

// -- ENCODE METHOD START --//

func (h *Header) encode() ([]byte, error) {
	buf := new(bytes.Buffer)
	fields := []any{h.ID, h.FLAGS, h.QDCOUNT, h.ANCOUNT, h.NSCOUNT, h.ARCOUNT}

	for _, field := range fields {
		if err := binary.Write(buf, binary.BigEndian, field); err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}

func (q *Question) encode() ([]byte, error) {
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

func (a *Answer) encode() ([]byte, error) {
	buf := new(bytes.Buffer)

	buf.Write(a.Name)

	_ = binary.Write(buf, binary.BigEndian, a.Type)
	_ = binary.Write(buf, binary.BigEndian, a.Class)
	_ = binary.Write(buf, binary.BigEndian, a.TTL)
	_ = binary.Write(buf, binary.BigEndian, a.Length)
	_ = binary.Write(buf, binary.BigEndian, a.Data)

	return buf.Bytes(), nil
}

func (dq *Query) Encode() ([]byte, error) {
	buf := new(bytes.Buffer)

	data, err := dq.Header.encode()
	if err != nil {
		return nil, err
	}
	buf.Write(data)
	data, err = dq.Questions.encode()
	if err != nil {
		return nil, err
	}
	buf.Write(data)

	// encoding each answer
	if dq.Answer != nil {
		for _, answer := range dq.Answer {
			data, err = answer.encode()
			if err != nil {
				return nil, err
			}
			buf.Write(data)
		}
	}
	return buf.Bytes(), nil
}

func encodeDomainName(name string) ([]byte, error) {
	// Convert domain name to DNS format
	var buf []byte
	labels := strings.Split(name, ".")

	for _, label := range labels {
		if len(label) == 0 {
			// If the label is empty, skip it
			continue
		}
		if len(label) > 63 {
			return nil, errors.New("label too long")
		}
		buf = append(buf, byte(len(label)))
		buf = append(buf, []byte(label)...)
	}
	buf = append(buf, 0) // Null byte to end the domain name
	return buf, nil
}

// -- ENCODE METHOD END -- //

// -- DECODE METHOD START -- //

func DecodeDNSQuery(data []byte) (*Query, error) {
	var dq Query
	header, err := decodeDNSHeader(data)
	if err != nil {
		return nil, err
	}
	dq.Header = header
	question, err := decodeDNSQuestion(data, 12)
	if err != nil {
		return nil, err
	}
	dq.Questions = question

	return &dq, nil
}

func decodeDNSHeader(data []byte) (*Header, error) {
	if len(data) < 12 {
		return nil, errors.New("malformed DNS header")
	}
	return &Header{
		ID:      binary.BigEndian.Uint16(data[0:2]),
		FLAGS:   binary.BigEndian.Uint16(data[2:4]),
		QDCOUNT: binary.BigEndian.Uint16(data[4:6]),
		ANCOUNT: binary.BigEndian.Uint16(data[6:8]),
		NSCOUNT: binary.BigEndian.Uint16(data[8:10]),
		ARCOUNT: binary.BigEndian.Uint16(data[10:12]),
	}, nil
}

func decodeDNSQuestion(data []byte, offset int) (*Question, error) {
	var q Question
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

func decodeDnsAnswer(data []byte) ([]*Answer, error) {
	if len(data) < 12 {
		return nil, errors.New("invalid DNS response")
	}

	anCount := binary.BigEndian.Uint16(data[6:8])
	if anCount == 0 {
		return nil, errors.New("no answers in DNS response")
	}

	offset := 12 // as we skipped header

	// Skip Question Section
	for offset < len(data) {
		length := int(data[offset])
		if length == 0 {
			offset++
			break
		}
		// handling compression pointer in question
		if length >= 0xC0 {
			offset += 2
			break
		}
		offset += 1 + length
	}

	// Skip QTYPE (2 bytes) and QCLASS (2 bytes)
	offset += 4
	answers := make([]*Answer, 0, anCount)

	for i := 0; i < int(anCount); i++ {

		if offset >= len(data) {
			return nil, errors.New("truncated DNS answer")
		}

		var nameBytes []byte // store original name bytes

		if data[offset] >= 0xC0 {
			nameBytes = make([]byte, 2)
			copy(nameBytes, data[offset:offset+2])
			offset += 2 // Compression pointer
		} else {
			nameEnd := offset
			for nameEnd < len(data) {
				length := int(data[nameEnd])
				if length == 0 {
					nameEnd++
					break
				}
				if length >= 0xC0 {
					nameEnd += 2 // Compression pointer
					break
				}
				nameEnd += 1 + length
			}
			nameBytes = make([]byte, nameEnd-offset)
			copy(nameBytes, data[offset:nameEnd])
			offset = nameEnd
		}

		if offset+10 > len(data) {
			return nil, errors.New("truncated DNS answer")
		}

		answerType := binary.BigEndian.Uint16(data[offset : offset+2])
		answerClass := binary.BigEndian.Uint16(data[offset+2 : offset+4])
		ttl := binary.BigEndian.Uint32(data[offset+4 : offset+8])
		dataLen := binary.BigEndian.Uint16(data[offset+8 : offset+10])
		offset += 10

		if offset+int(dataLen) > len(data) {
			return nil, errors.New("incomplete DNS answer data")
		}

		answerData := make([]byte, dataLen)
		copy(answerData, data[offset:offset+int(dataLen)])

		answer := &Answer{
			Name:   nameBytes,
			Type:   answerType,
			Class:  answerClass,
			TTL:    ttl,
			Length: dataLen,
			Data:   answerData,
		}

		answers = append(answers, answer)

		offset += int(dataLen)
	}

	return answers, nil
}

// -- DECODE METHOD END -- //
