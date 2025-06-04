package cache

import "time"

type RecordType uint16

const (
	RecordTypeA        RecordType = 1
	RecordTypeAAAA     RecordType = 28
	RecordTypeCNAME    RecordType = 5
	RecordTypeMX       RecordType = 15
	RecordTypeNS       RecordType = 2
	RecordTypePTR      RecordType = 12
	RecordTypeSOA      RecordType = 6
	RecordTypeSRV      RecordType = 33
	RecordTypeTXT      RecordType = 16
	RecordTypeCAA      RecordType = 257
	RecordTypeNXDomain RecordType = 65535
)

type Record struct {
	Type      RecordType
	ExpiresAt time.Time // CreatedAt + TTL
	Data      []byte    // record specific data
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
