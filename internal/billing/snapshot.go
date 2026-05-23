package billing

import (
	"encoding/json"
	"time"

	"github.com/hejunqiu/ds-code/internal/llm"
	"github.com/hejunqiu/ds-code/internal/session"
)

// CurrencyCNY is the billing currency code stored in snapshots.
const CurrencyCNY = "CNY"

// PriceSnapshot freezes per-million CNY rates used for one estimate.
type PriceSnapshot struct {
	Currency           string    `json:"currency"`
	ModelID            string    `json:"model_id"`
	PriceTableVersion  string    `json:"price_table_version"`
	InputPerMillion    float64   `json:"input_per_million"`
	OutputPerMillion   float64   `json:"output_per_million"`
	CacheHitPerMillion float64   `json:"cache_hit_per_million"`
	CapturedAt         time.Time `json:"captured_at"`
}

// SnapshotForModel copies current list prices for modelID into a snapshot.
func SnapshotForModel(modelID string) PriceSnapshot {
	p, modelID := lookupPrices(modelID)
	return PriceSnapshot{
		Currency:           CurrencyCNY,
		ModelID:            modelID,
		PriceTableVersion:  currentPriceTableVersion(),
		InputPerMillion:    p.Input,
		OutputPerMillion:   p.Output,
		CacheHitPerMillion: p.CacheHit,
		CapturedAt:         time.Now().UTC(),
	}
}

// MarshalSnapshot JSON-encodes a price snapshot for persistence.
func MarshalSnapshot(s PriceSnapshot) string {
	raw, err := json.Marshal(s)
	if err != nil {
		return ""
	}
	return string(raw)
}

// ParseSnapshot decodes a persisted snapshot; empty jsonStr yields current prices for modelID.
// Corrupt JSON yields a zero-rate snapshot (cost estimate 0) rather than current prices.
func ParseSnapshot(modelID, jsonStr string) PriceSnapshot {
	if jsonStr == "" {
		return SnapshotForModel(modelID)
	}
	var s PriceSnapshot
	if err := json.Unmarshal([]byte(jsonStr), &s); err != nil || s.ModelID == "" {
		return PriceSnapshot{Currency: CurrencyCNY, ModelID: modelID}
	}
	return s
}

// EstimateCNYFromSnapshot computes cost in CNY for one usage record.
func EstimateCNYFromSnapshot(s PriceSnapshot, u llm.Usage) float64 {
	inBill := u.PromptTokens - u.PromptCacheHitTokens
	if inBill < 0 {
		inBill = 0
	}
	return float64(inBill)/1e6*s.InputPerMillion +
		float64(u.PromptCacheHitTokens)/1e6*s.CacheHitPerMillion +
		float64(u.CompletionTokens)/1e6*s.OutputPerMillion
}

// EstimateCNYFromSnapshotTotals computes cost from cumulative snapshot fields.
func EstimateCNYFromSnapshotTotals(s PriceSnapshot, snap session.UsageSnapshot) float64 {
	inBill := snap.PromptTokensTotal - snap.PromptCacheHitTokensTotal
	if inBill < 0 {
		inBill = 0
	}
	return float64(inBill)/1e6*s.InputPerMillion +
		float64(snap.PromptCacheHitTokensTotal)/1e6*s.CacheHitPerMillion +
		float64(snap.CompletionTokensTotal)/1e6*s.OutputPerMillion
}
