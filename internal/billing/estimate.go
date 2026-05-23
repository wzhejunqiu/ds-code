package billing

import (
	"fmt"

	"github.com/hejunqiu/ds-code/internal/session"
)

// PerMillion holds CNY price per 1M tokens (static table; update when vendor pricing changes).
type PerMillion struct {
	Input    float64
	Output   float64
	CacheHit float64
}

// EstimateCNY returns cumulative session cost in CNY from usage totals and current prices.
func EstimateCNY(model string, snap session.UsageSnapshot) float64 {
	return EstimateCNYFromSnapshotTotals(SnapshotForModel(model), snap)
}

// FormatCNY formats a yuan amount for the status bar.
func FormatCNY(cny float64) string {
	if cny < 0.01 {
		return fmt.Sprintf("¥%.4f", cny)
	}
	return fmt.Sprintf("¥%.3f", cny)
}
