// A separate channel for log events from the DNS server to the UI
// its seperated because the UI should not be blocked by other global events

package channels

var LogEventChannel = make(chan Event, 10)
