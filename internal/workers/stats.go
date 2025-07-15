package workers

import (
	"fmt"
	"runtime"
	"strings"
)

type WorkerPoolStats struct {
	ActiveWalkers   int32
	ActiveTesters   int32
	DomainCount     int32
	TotalGoroutines int32
	ResultsObtained int32
	ResultsToTest   int32
	PathsToWalk     int32
}

func (wp *WorkerPool) GetStats() WorkerPoolStats {
	return WorkerPoolStats{
		ActiveWalkers:   wp.activeWalkers.Load(),
		ActiveTesters:   wp.activeTesters.Load(),
		DomainCount:     int32(wp.GetDomainCount()),
		TotalGoroutines: int32(runtime.NumGoroutine()),
		ResultsObtained: int32(len(wp.resultsCache.ResultsData)),
		ResultsToTest:   int32(len(wp.toTestChan)),
		PathsToWalk:     int32(len(wp.toWalkChan)),
	}
}

func (wp *WorkerPool) GetStatsString(termWidth int) string {
	stats := wp.GetStats()

	var lines []string
	lines = append(lines, fmt.Sprintf("🚶 Active Walkers: %d", stats.ActiveWalkers))
	lines = append(lines, fmt.Sprintf("🧪 Active Testers: %d", stats.ActiveTesters))
	lines = append(lines, fmt.Sprintf("🌐 Domain Count: %d", stats.DomainCount))
	lines = append(lines, fmt.Sprintf("⚡ Total Goroutines: %d", stats.TotalGoroutines))
	lines = append(lines, fmt.Sprintf("✅ Results Obtained: %d", stats.ResultsObtained))
	lines = append(lines, fmt.Sprintf("📋 Results To Test: %d", stats.ResultsToTest))
	lines = append(lines, fmt.Sprintf("📁 Paths To Walk: %d", stats.PathsToWalk))

	return strings.Join(lines, "\n")
}
