package workers

import "runtime"

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
