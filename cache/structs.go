package cache

import (
	"sync"
	"time"
)

type entry struct {
	key    string
	record *Record // DNS record
	prev   *entry
	next   *entry
}

type LRUCache struct {
	capacity  int
	items     map[string]*entry
	head      *entry // most recently used
	tail      *entry // least recently used
	mutex     sync.RWMutex
	cleanUpCh chan struct{} // signal to clean up expired entries

	// prefetch
	prefetch      chan struct{}
	slidingWindow time.Duration
	topCount      int
}

type domainMetrics struct {
	domain                string
	numberOfTimesAccessed int
	lastAccessed          time.Time
}
