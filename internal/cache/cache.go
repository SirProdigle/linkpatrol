package cache

/*
TODO later one we might want to add a mechanism to flush cache to disk
when it reaches a certain size and then dedupe it all back into memory at the end
*/
import (
	"fmt"
	"strings"
	"sync"

	"github.com/sirprodigle/linkpatrol/internal/logger"
)

type CacheEntry struct {
	URL    string
	Status CacheEntryStatus
	Error  string
}

type Cache struct {
	entries    map[string]CacheEntry
	mutex      sync.RWMutex
	maxEntries int
	testing    map[string]bool // URLs currently being tested
	testMutex  sync.RWMutex    // Separate mutex for testing map
}

type CacheEntryStatus int

const (
	Live CacheEntryStatus = iota
	Timeout
	Dead
)

type CacheOption func(*Cache)

func WithMaxEntries(maxEntries int) CacheOption {
	return func(c *Cache) {
		c.maxEntries = maxEntries
	}
}

func NewCache(opts ...CacheOption) *Cache {
	cache := &Cache{
		entries:    make(map[string]CacheEntry, 2000),
		mutex:      sync.RWMutex{},
		maxEntries: 0,
		testing:    make(map[string]bool),
		testMutex:  sync.RWMutex{},
	}

	for _, opt := range opts {
		opt(cache)
	}

	return cache
}

func (c *Cache) Add(url string, status CacheEntryStatus, errors ...string) error {
	if url == "" {
		return fmt.Errorf("url is empty")
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Check if cache is full
	if c.maxEntries > 0 && len(c.entries) >= c.maxEntries {
		return fmt.Errorf("cache is full")
	}

	errorMsg := ""
	if len(errors) > 0 {
		errorMsg = strings.Join(errors, "\n")
	}

	c.entries[url] = CacheEntry{
		URL:    url,
		Status: status,
		Error:  errorMsg,
	}

	return nil
}

func (c *Cache) Get(url string) *CacheEntry {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	entry, ok := c.entries[url]
	if !ok {
		return nil
	}

	return &entry
}

// IsBeingTested checks if a URL is currently being tested
func (c *Cache) IsBeingTested(url string) bool {
	c.testMutex.RLock()
	defer c.testMutex.RUnlock()
	return c.testing[url]
}

// StartTesting marks a URL as being tested. Returns false if already being tested.
func (c *Cache) StartTesting(url string) bool {
	c.testMutex.Lock()
	defer c.testMutex.Unlock()

	if c.testing[url] {
		return false // Already being tested
	}

	c.testing[url] = true
	return true
}

// FinishTesting marks a URL as no longer being tested
func (c *Cache) FinishTesting(url string) {
	c.testMutex.Lock()
	defer c.testMutex.Unlock()
	delete(c.testing, url)
}

// Clear removes all entries
func (c *Cache) Clear() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.entries = make(map[string]CacheEntry)
}

func (c *Cache) HasFailures() bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	for _, entry := range c.entries {
		if entry.Status == Dead || entry.Status == Timeout {
			return true
		}
	}
	return false
}

func (c *Cache) GetFailureCount() (deadCount, timeoutCount int) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	for _, entry := range c.entries {
		switch entry.Status {
		case Dead:
			deadCount++
		case Timeout:
			timeoutCount++
		}
	}
	return deadCount, timeoutCount
}

// PrettyPrint displays the cache contents using the standardized logger
func (c *Cache) PrettyPrint(log *logger.Logger) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if len(c.entries) == 0 {
		log.Info("No cache entries to display")
		return
	}

	// Convert cache entries to display format
	displayEntries := make([]logger.DisplayEntry, 0, len(c.entries))
	for _, entry := range c.entries {
		displayEntry := c.formatEntryForDisplay(entry)
		displayEntries = append(displayEntries, displayEntry)
	}

	log.CacheTable(displayEntries)
}

// formatEntryForDisplay converts a cache entry to a display-ready format
func (c *Cache) formatEntryForDisplay(entry CacheEntry) logger.DisplayEntry {
	var color, emoji, status string
	switch entry.Status {
	case Live:
		color = "\033[32m" // Green
		emoji = "✅"
		status = "LIVE"
	case Dead:
		color = "\033[31m" // Red
		emoji = "❌"
		status = "DEAD"
	case Timeout:
		color = "\033[33m" // Yellow
		emoji = "⏰"
		status = "TIMEOUT"
	default:
		color = "\033[34m" // Blue
		emoji = "❓"
		status = "UNKNOWN"
	}

	return logger.DisplayEntry{
		URL:    entry.URL,
		Status: status,
		Emoji:  emoji,
		Error:  entry.Error,
		Color:  color,
	}
}
