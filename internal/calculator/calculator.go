package calculator

import (
	"log"
	"sort"
	"sync"
)

// TokenUsage is the token count of one request (or an aggregate of requests).
//
// CacheWriteTokens is the total cache-write count (the transcript's
// cache_creation_input_tokens). CacheWrite1hTokens is the part of that total
// written to the 1-hour cache (cache_creation.ephemeral_1h_input_tokens); the
// remainder is priced at the 5-minute rate. Leave CacheWrite1hTokens at zero
// when the transcript carries no per-TTL split.
type TokenUsage struct {
	InputTokens        int64
	OutputTokens       int64
	CacheReadTokens    int64
	CacheWriteTokens   int64
	CacheWrite1hTokens int64
}

type CostBreakdown struct {
	InputCost      float64
	OutputCost     float64
	CacheReadCost  float64
	CacheWriteCost float64
	TotalCost      float64
}

// Calculate prices usage at the list rates for model. Unknown models are
// priced at Fallback and recorded (see UnknownModels); the first sighting of
// each unknown model id is logged.
func Calculate(model string, usage TokenUsage) CostBreakdown {
	rates, known := LookupRates(model)
	if !known {
		recordUnknown(model)
	}

	write1h := usage.CacheWrite1hTokens
	if write1h > usage.CacheWriteTokens {
		write1h = usage.CacheWriteTokens
	}
	write5m := usage.CacheWriteTokens - write1h

	cb := CostBreakdown{
		InputCost:     float64(usage.InputTokens) / 1_000_000 * rates.InputPerMToken,
		OutputCost:    float64(usage.OutputTokens) / 1_000_000 * rates.OutputPerMToken,
		CacheReadCost: float64(usage.CacheReadTokens) / 1_000_000 * rates.CacheReadPerMToken,
		CacheWriteCost: float64(write5m)/1_000_000*rates.CacheWritePerMToken +
			float64(write1h)/1_000_000*rates.CacheWrite1hPerMToken,
	}
	cb.TotalCost = cb.InputCost + cb.OutputCost + cb.CacheReadCost + cb.CacheWriteCost
	return cb
}

func (u TokenUsage) Total() int64 {
	return u.InputTokens + u.OutputTokens + u.CacheReadTokens + u.CacheWriteTokens
}

// --- Unknown-model accounting ---

var (
	unknownMu     sync.Mutex
	unknownCounts = map[string]int64{}
)

func recordUnknown(model string) {
	unknownMu.Lock()
	defer unknownMu.Unlock()
	if unknownCounts[model] == 0 {
		log.Printf("Warning: unknown model %q priced at fallback rates (%s)", model, FallbackFamily)
	}
	unknownCounts[model]++
}

// UnknownModel is one model id that matched no rates entry in this process.
type UnknownModel struct {
	Model string
	Count int64
}

// UnknownModels returns every model id priced at Fallback by Calculate in this
// process, with how many times, sorted by count descending.
func UnknownModels() []UnknownModel {
	unknownMu.Lock()
	defer unknownMu.Unlock()
	out := make([]UnknownModel, 0, len(unknownCounts))
	for m, n := range unknownCounts {
		out = append(out, UnknownModel{Model: m, Count: n})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Model < out[j].Model
	})
	return out
}

// ResetUnknownModels clears the unknown-model counters (for tests).
func ResetUnknownModels() {
	unknownMu.Lock()
	defer unknownMu.Unlock()
	unknownCounts = map[string]int64{}
}
