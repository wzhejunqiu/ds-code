package manager

import "sync"

var (
	trackedMu sync.Mutex
	tracked   map[string]map[string]struct{} // jobsDir -> job ID
)

func trackJob(jobsDir, id string) {
	trackedMu.Lock()
	defer trackedMu.Unlock()
	if tracked == nil {
		tracked = make(map[string]map[string]struct{})
	}
	if tracked[jobsDir] == nil {
		tracked[jobsDir] = make(map[string]struct{})
	}
	tracked[jobsDir][id] = struct{}{}
}

func untrackJob(jobsDir, id string) {
	trackedMu.Lock()
	defer trackedMu.Unlock()
	if ids, ok := tracked[jobsDir]; ok {
		delete(ids, id)
		if len(ids) == 0 {
			delete(tracked, jobsDir)
		}
	}
}

func isTracked(jobsDir, id string) bool {
	trackedMu.Lock()
	defer trackedMu.Unlock()
	ids, ok := tracked[jobsDir]
	if !ok {
		return false
	}
	_, ok = ids[id]
	return ok
}
