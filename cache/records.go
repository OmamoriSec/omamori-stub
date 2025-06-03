package cache

import "time"

type RecordType string

const (
	RecordTypeA        RecordType = "A"
	RecordTypeAAAA     RecordType = "AAAA"
	RecordTypeCNAME    RecordType = "CNAME"
	RecordTypeMX       RecordType = "MX"
	RecordTypeNS       RecordType = "NS"
	RecordTypePTR      RecordType = "PTR"
	RecordTypeSOA      RecordType = "SOA"
	RecordTypeSRV      RecordType = "SRV"
	RecordTypeTXT      RecordType = "TXT"
	RecordTypeCAA      RecordType = "CAA"
	RecordTypeNXDomain RecordType = "NXDomain"
)

type Record struct {
	Type      RecordType
	ExpiresAt time.Time   // CreatedAt + TTL
	Data      interface{} // record specific data
}

type ARecord struct {
	IPAddress string
}

type AAAARecord struct {
	IPAddress string
}

type CNAMERecord struct {
	Target string
}

type MXRecord struct {
	Priority uint16
	Target   string // Mail server hostname
}

type NSRecord struct {
	NameServer string
}

type PTRRecord struct {
	PTRDName string
}

type SOARecord struct {
	MName        string
	RName        string
	SerialNumber string
	Refresh      uint32
	Retry        uint32
	Expire       uint32
	Minimum      uint32
}

type TXTRecord struct {
	Text []string
}

type CAARecord struct {
	Flag  uint8
	Tag   string
	Value string
}

type SRVRecord struct {
	Priority uint16
	Weight   uint16
	Port     uint16
	Target   string
}

type NXDomainRecord struct{}

type Key struct {
	Domain string
	Type   RecordType
}
