package cache

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

var DnsCache = DNSCache(1000)

func (c *LRUCache) removeEntry(e *entry) {
	// updating the next and prev pointers
	if e.prev != nil {
		e.prev.next = e.next
	} else {
		c.head = e.next
	}

	// update next node
	if e.next != nil {
		e.next.prev = e.prev
	} else {
		c.tail = e.prev
	}

	e.prev = nil // clear the prev pointer
	e.next = nil // clear the next pointer
}

func (c *LRUCache) removeTail() {
	if c.tail == nil {
		return
	}

	delete(c.items, c.tail.key)
	c.removeEntry(c.tail)
}

func (c *LRUCache) moveToFront(e *entry) {
	if c.head == e {
		return // entry already at the front
	}

	c.removeEntry(e) // remove the entry from the current position

	// adding it to the front
	e.next = c.head
	e.prev = nil
	if c.head != nil {
		c.head.prev = e
	}
	c.head = e

	if c.tail == nil { // if tail removed, update it
		c.tail = e
	}
}

// cleanup periodically checks and removes the expired entries
func (c *LRUCache) startCleanUp() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.removeExpired()
		case <-c.cleanUpCh:
			// cleanup signal received
			return
		}
	}
}

func (c *LRUCache) removeExpired() {
	now := time.Now()
	c.mutex.Lock()
	defer c.mutex.Unlock()

	for key, e := range c.items {
		if now.After(e.record.ExpiresAt) {
			delete(c.items, key)
			c.removeEntry(e)
		}
	}
}

func NormalizeDomain(domain string) string {
	return strings.ToLower(domain) // TODO!: To add more rigorous normalization
}

func (k Key) String() string {
	return strconv.Itoa(int(k.Type)) + ":" + k.Domain
}

func (c *LRUCache) PrintCacheContents() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	fmt.Println("Cache contents:")
	for key, entry := range c.items {
		fmt.Printf("Key: %s, Record: %v\n", key, entry.record)
	}
}
