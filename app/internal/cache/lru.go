package cache

import (
	"time"
)

type Cache interface {
	Get(domain string, recordType uint16) (*Record, bool)

	Set(domain string, record *Record)

	Remove(domain string, recordType uint16)

	Close()
}

func NewCacheKey(domain string, recordType RecordType) Key {
	return Key{
		Domain: NormalizeDomain(domain),
		Type:   recordType,
	}
}

func DNSCache(capacity int) *LRUCache {
	cache := &LRUCache{
		capacity:  capacity,
		items:     make(map[string]*entry, capacity),
		cleanUpCh: make(chan struct{}),
	}
	go cache.startCleanUp()

	return cache
}

func (c *LRUCache) Get(domain string, recordType uint16) (*Record, bool) {
	key := NewCacheKey(domain, RecordType(recordType)).String()
	
	c.mutex.RLock()
	e, found := c.items[key]
	c.mutex.RUnlock()

	if !found {
		return nil, false
	}

	// check if the entry has expired
	if time.Now().After(e.record.ExpiresAt) {
		c.Remove(domain, recordType)
		return nil, false
	}

	c.mutex.Lock() // acquiring a Write lock for the updates
	defer c.mutex.Unlock()

	if _, stillExists := c.items[key]; !stillExists {
		return nil, false
	}

	// move to front
	c.moveToFront(e)

	return e.record, true
}

func (c *LRUCache) Set(domain string, record *Record) {
	key := NewCacheKey(domain, record.Type).String()

	c.mutex.Lock()
	defer c.mutex.Unlock()

	// check if the key already exists
	if e, found := c.items[key]; found {
		e.record = record
		c.moveToFront(e)
		return
	}

	// new entry
	e := &entry{
		key:    key,
		record: record,
	}
	c.items[key] = e // adding to the map

	// adding to the linked list
	if c.head == nil {
		// first entry
		c.head = e
		c.tail = e
	} else {
		// adding to the front
		e.next = c.head
		c.head.prev = e
		c.head = e
	}

	// check if the cache is full, remove the lru entry
	if len(c.items) > c.capacity {
		c.removeTail()
	}
}

func (c *LRUCache) Remove(domain string, recordType uint16) {
	key := NewCacheKey(domain, RecordType(recordType)).String()

	c.mutex.Lock()
	defer c.mutex.Unlock()

	e, found := c.items[key]
	if !found {
		return
	}

	delete(c.items, key) // remove from the map
	c.removeEntry(e)     // remove from the linked list
}

func (c *LRUCache) Close() {
	close(c.cleanUpCh)
}

// TODO! making cache persistent
