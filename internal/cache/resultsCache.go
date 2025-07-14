package cache

import (
	"sync"

	"github.com/sirprodigle/linkpatrol/internal/logger"
)

type CacheEntry struct {
	URL    string
	Status CacheEntryStatus
	Error  string
}

type CacheEntryStatus int

const (
	Live CacheEntryStatus = iota
	Timeout
	Dead
	BotDetected
)

type ResultsCache struct {
	ResultsData  map[string]CacheEntry
	ResultsMutex sync.RWMutex
	ResultsChan  <-chan CacheEntry
}

func NewResultsCache(resultsReadChan <-chan CacheEntry) *ResultsCache {
	return &ResultsCache{
		ResultsData: make(map[string]CacheEntry, 1000),
		ResultsChan: resultsReadChan,
	}
}

func (c *ResultsCache) HasResult(url string) bool {
	c.ResultsMutex.RLock()
	defer c.ResultsMutex.RUnlock()

	_, ok := c.ResultsData[url]
	return ok
}

func (c *ResultsCache) GetResult(url string) CacheEntry {
	c.ResultsMutex.RLock()
	defer c.ResultsMutex.RUnlock()

	return c.ResultsData[url]
}

func (c *ResultsCache) DoLoop() {
	go func() {
		for result := range c.ResultsChan {
			c.ResultsMutex.Lock()
			c.ResultsData[result.URL] = result
			c.ResultsMutex.Unlock()
		}
	}()
}

func (c *ResultsCache) PrettyPrint(logger *logger.Logger) {
	logger.StartSection("Results")
	for url, result := range c.ResultsData {
		logger.Debug("%s -> %s", url, result.Status)
	}
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
