package cache

import (
	"sync"
)

type CacheEntry struct {
	URL    string
	Status CacheEntryStatus
	Error  string
}

//go:generate stringer -type=CacheEntryStatus
type CacheEntryStatus int

const (
	Live CacheEntryStatus = iota
	Timeout
	Dead
	Bot
	Ignore
)

type ResultsCache struct {
	ResultsData  map[string]CacheEntry
	ClaimedURLs  map[string]bool
	ResultsMutex sync.RWMutex
	ResultsChan  <-chan CacheEntry
}

func NewResultsCache(resultsReadChan <-chan CacheEntry) *ResultsCache {
	return &ResultsCache{
		ResultsData: make(map[string]CacheEntry, 1000),
		ClaimedURLs: make(map[string]bool, 1000),
		ResultsChan: resultsReadChan,
	}
}

func (c *ResultsCache) HasResult(url string) bool {
	c.ResultsMutex.RLock()
	defer c.ResultsMutex.RUnlock()

	_, ok := c.ResultsData[url]
	return ok
}

// TryClaim attempts to claim a URL for processing. Returns true if claimed successfully, false if already processed/claimed.
func (c *ResultsCache) TryClaim(url string) bool {
	c.ResultsMutex.Lock()
	defer c.ResultsMutex.Unlock()

	// Check if already processed
	if _, exists := c.ResultsData[url]; exists {
		return false
	}

	// Check if already claimed
	if _, claimed := c.ClaimedURLs[url]; claimed {
		return false
	}

	// Claim it
	c.ClaimedURLs[url] = true
	return true
}

func (c *ResultsCache) GetResult(url string) CacheEntry {
	c.ResultsMutex.RLock()
	defer c.ResultsMutex.RUnlock()

	return c.ResultsData[url]
}

func (c *ResultsCache) GetResults() []CacheEntry {
	c.ResultsMutex.RLock()
	defer c.ResultsMutex.RUnlock()
	results := make([]CacheEntry, 0, len(c.ResultsData))
	for _, result := range c.ResultsData {
		results = append(results, result)
	}
	return results
}

func (c *ResultsCache) DoLoop() {
	go func() {
		for result := range c.ResultsChan {
			c.ResultsMutex.Lock()
			c.ResultsData[result.URL] = result
			// Remove from claimed when we have a result
			delete(c.ClaimedURLs, result.URL)
			c.ResultsMutex.Unlock()
		}
	}()
}

func (c *ResultsCache) HasFailures() bool {
	for _, result := range c.ResultsData {
		if result.Status == Dead || result.Status == Timeout {
			return true
		}
	}
	return false
}

func (c *ResultsCache) GetFailureCount() (int, int) {
	deadCount := 0
	timeoutCount := 0
	for _, result := range c.ResultsData {
		if result.Status == Dead {
			deadCount++
		}
		if result.Status == Timeout {
			timeoutCount++
		}
	}
	return deadCount, timeoutCount
}

func (c *ResultsCache) CleanUpIgnoredResults() {
	for url, result := range c.ResultsData {
		if result.Status == Ignore {
			delete(c.ResultsData, url)
		}
	}
}
