package calculator

// ModelRates is the list price of one model family in USD per million tokens.
//
// CacheWritePerMToken is the 5-minute cache write rate (1.25x input);
// CacheWrite1hPerMToken is the 1-hour cache write rate (2x input). Claude Code
// main sessions write the 1-hour cache, subagents the 5-minute cache, so both
// are needed to price a transcript correctly.
type ModelRates struct {
	Family                string
	InputPerMToken        float64
	OutputPerMToken       float64
	CacheReadPerMToken    float64
	CacheWritePerMToken   float64 // 5m cache write
	CacheWrite1hPerMToken float64 // 1h cache write
}

// Source of every figure below unless a comment says otherwise:
//
//	https://platform.claude.com/docs/en/about-claude/pricing  (fetched 2026-09-24)
//	cross-checked against https://claude.com/pricing           (fetched 2026-09-24)
//
// Rates maps model-id prefixes to their list pricing. A model id is matched by
// prefix ("claude-haiku-4-5-20251001" matches "claude-haiku-4-5"), so more
// specific prefixes MUST come before shorter ones within a family:
// "claude-opus-5-5" before "claude-opus-5", "claude-fable-5-1" before
// "claude-fable-5". TestRatesOrdering enforces this.
//
// Standard multipliers on the pricing page: 5m cache write = 1.25x input,
// 1h cache write = 2x input, cache read = 0.1x input, except cache read is
// 0.025x on Fable 5.1 / Mythos 5.1 and 0.05x on Opus 5.5.
//
// Unknown models are NOT matched here; see Fallback and LookupRates.
var Rates = []ModelRates{
	// Claude Fable 5.1 — $10 / $12.50 / $20 / $0.25 / $50 (platform.claude.com pricing, 2026-09-24)
	{Family: "claude-fable-5-1", InputPerMToken: 10.00, OutputPerMToken: 50.00, CacheReadPerMToken: 0.25, CacheWritePerMToken: 12.50, CacheWrite1hPerMToken: 20.00},
	// Claude Mythos 5.1 (limited availability) — same row as Fable 5.1 (platform.claude.com pricing, 2026-09-24)
	{Family: "claude-mythos-5-1", InputPerMToken: 10.00, OutputPerMToken: 50.00, CacheReadPerMToken: 0.25, CacheWritePerMToken: 12.50, CacheWrite1hPerMToken: 20.00},
	// Claude Fable 5 — $10 / $12.50 / $20 / $1 / $50 (platform.claude.com pricing, 2026-09-24)
	{Family: "claude-fable-5", InputPerMToken: 10.00, OutputPerMToken: 50.00, CacheReadPerMToken: 1.00, CacheWritePerMToken: 12.50, CacheWrite1hPerMToken: 20.00},
	// Claude Mythos 5 (limited availability) — same row as Fable 5 (platform.claude.com pricing, 2026-09-24)
	{Family: "claude-mythos-5", InputPerMToken: 10.00, OutputPerMToken: 50.00, CacheReadPerMToken: 1.00, CacheWritePerMToken: 12.50, CacheWrite1hPerMToken: 20.00},

	// Claude Opus 5.5 — $4 / $5 / $8 / $0.20 / $20 (platform.claude.com pricing, 2026-09-24)
	{Family: "claude-opus-5-5", InputPerMToken: 4.00, OutputPerMToken: 20.00, CacheReadPerMToken: 0.20, CacheWritePerMToken: 5.00, CacheWrite1hPerMToken: 8.00},
	// Claude Opus 5 — $5 / $6.25 / $10 / $0.50 / $25 (platform.claude.com pricing, 2026-09-24)
	{Family: "claude-opus-5", InputPerMToken: 5.00, OutputPerMToken: 25.00, CacheReadPerMToken: 0.50, CacheWritePerMToken: 6.25, CacheWrite1hPerMToken: 10.00},

	// Claude Opus 4.8 / 4.7 / 4.6 / 4.5 — $5 / $6.25 / $10 / $0.50 / $25 (platform.claude.com pricing, 2026-09-24)
	{Family: "claude-opus-4-8", InputPerMToken: 5.00, OutputPerMToken: 25.00, CacheReadPerMToken: 0.50, CacheWritePerMToken: 6.25, CacheWrite1hPerMToken: 10.00},
	{Family: "claude-opus-4-7", InputPerMToken: 5.00, OutputPerMToken: 25.00, CacheReadPerMToken: 0.50, CacheWritePerMToken: 6.25, CacheWrite1hPerMToken: 10.00},
	{Family: "claude-opus-4-6", InputPerMToken: 5.00, OutputPerMToken: 25.00, CacheReadPerMToken: 0.50, CacheWritePerMToken: 6.25, CacheWrite1hPerMToken: 10.00},
	{Family: "claude-opus-4-5", InputPerMToken: 5.00, OutputPerMToken: 25.00, CacheReadPerMToken: 0.50, CacheWritePerMToken: 6.25, CacheWrite1hPerMToken: 10.00},
	// Claude Opus 4.1 (retired) — $15 / $18.75 / $30 / $1.50 / $75 (platform.claude.com pricing, 2026-09-24)
	{Family: "claude-opus-4-1", InputPerMToken: 15.00, OutputPerMToken: 75.00, CacheReadPerMToken: 1.50, CacheWritePerMToken: 18.75, CacheWrite1hPerMToken: 30.00},
	// Claude Opus 4 (retired; id "claude-opus-4-20250514") — $15 / $18.75 / $30 / $1.50 / $75 (platform.claude.com pricing, 2026-09-24)
	{Family: "claude-opus-4", InputPerMToken: 15.00, OutputPerMToken: 75.00, CacheReadPerMToken: 1.50, CacheWritePerMToken: 18.75, CacheWrite1hPerMToken: 30.00},

	// Claude Sonnet 5 — $2 / $2.50 / $4 / $0.20 / $10; the launch price is now standard (platform.claude.com pricing, 2026-09-24)
	{Family: "claude-sonnet-5", InputPerMToken: 2.00, OutputPerMToken: 10.00, CacheReadPerMToken: 0.20, CacheWritePerMToken: 2.50, CacheWrite1hPerMToken: 4.00},
	// Claude Sonnet 4.6 / 4.5 / 4 (retired) — all $3 / $3.75 / $6 / $0.30 / $15 (platform.claude.com pricing, 2026-09-24)
	{Family: "claude-sonnet-4", InputPerMToken: 3.00, OutputPerMToken: 15.00, CacheReadPerMToken: 0.30, CacheWritePerMToken: 3.75, CacheWrite1hPerMToken: 6.00},

	// Claude Haiku 4.5 — $1 / $1.25 / $2 / $0.10 / $5 (platform.claude.com pricing, 2026-09-24)
	{Family: "claude-haiku-4-5", InputPerMToken: 1.00, OutputPerMToken: 5.00, CacheReadPerMToken: 0.10, CacheWritePerMToken: 1.25, CacheWrite1hPerMToken: 2.00},
	// UNVERIFIED: no Haiku 4.x other than 4.5 is on the public pricing page (2026-09-24).
	// Kept as a catch-all for any other "claude-haiku-4*" id at Haiku 4.5 rates.
	{Family: "claude-haiku-4", InputPerMToken: 1.00, OutputPerMToken: 5.00, CacheReadPerMToken: 0.10, CacheWritePerMToken: 1.25, CacheWrite1hPerMToken: 2.00},
	// Claude Haiku 3.5 (retired) — $0.80 / $1 / $1.60 / $0.08 / $4 (platform.claude.com pricing, 2026-09-24)
	{Family: "claude-haiku-3-5", InputPerMToken: 0.80, OutputPerMToken: 4.00, CacheReadPerMToken: 0.08, CacheWritePerMToken: 1.00, CacheWrite1hPerMToken: 1.60},
}

// FallbackFamily is the Family reported for models that match no entry in Rates.
const FallbackFamily = "unknown-model (fallback: Sonnet 4.x rates)"

// Fallback is used for model ids that match nothing in Rates. It carries the
// Sonnet 4.x list rates purely as a placeholder; the Family string makes the
// fallback visible in the API/UI, and Calculate counts every use so
// `cctrack status` can report unknown models. Never add a real model here.
var Fallback = ModelRates{
	Family:                FallbackFamily,
	InputPerMToken:        3.00,
	OutputPerMToken:       15.00,
	CacheReadPerMToken:    0.30,
	CacheWritePerMToken:   3.75,
	CacheWrite1hPerMToken: 6.00,
}

// LookupRates returns the rates for the first family whose prefix matches
// model, and whether a match was found. On no match it returns Fallback, false.
func LookupRates(model string) (*ModelRates, bool) {
	for i := range Rates {
		if len(model) >= len(Rates[i].Family) && model[:len(Rates[i].Family)] == Rates[i].Family {
			return &Rates[i], true
		}
	}
	return &Fallback, false
}

// GetRates returns the rates for model, or Fallback when the model is unknown.
// Callers that need to know whether the model was recognised should use
// LookupRates.
func GetRates(model string) *ModelRates {
	r, _ := LookupRates(model)
	return r
}
