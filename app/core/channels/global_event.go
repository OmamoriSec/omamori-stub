package channels

type EventType string

const (
	StartDnsServer EventType = "START_DNS_SERVER"
	StopDnsServer  EventType = "STOP_DNS_SERVER"
	StartDOHServer EventType = "START_DOH_SERVER"
	StopDOHServer  EventType = "STOP_DOH_SERVER"
	UpdateConfig   EventType = "UPDATE_CONFIG"
	UpdateSiteList EventType = "UPDATE_SITE_LIST"

	ConfigureSystemDNS EventType = "CONFIGURE_SYSTEM_DNS"
	RestoreSystemDNS   EventType = "RESTORE_SYSTEM_DNS"

	Error EventType = "ERROR"
	Log   EventType = "LOG"
)

type Event struct {
	Type    EventType
	Payload interface{} // can be nil or config struct or site map i.e. anything else
}

var GlobalEventChannel = make(chan Event, 10)
