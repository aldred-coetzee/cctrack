package calculator

import (
	"math"
	"strings"
	"testing"
)

const mtok = 1_000_000

func approx(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

// One row per family: $ for 1,000,000 tokens of each class must equal the list
// figure on https://platform.claude.com/docs/en/about-claude/pricing (2026-09-24).
func TestListPricePerMillionTokens(t *testing.T) {
	cases := []struct {
		model                                      string
		family                                     string
		input, write5m, write1h, cacheRead, output float64
	}{
		{"claude-fable-5-1", "claude-fable-5-1", 10.00, 12.50, 20.00, 0.25, 50.00},
		{"claude-mythos-5-1", "claude-mythos-5-1", 10.00, 12.50, 20.00, 0.25, 50.00},
		{"claude-fable-5", "claude-fable-5", 10.00, 12.50, 20.00, 1.00, 50.00},
		{"claude-mythos-5", "claude-mythos-5", 10.00, 12.50, 20.00, 1.00, 50.00},
		{"claude-opus-5-5", "claude-opus-5-5", 4.00, 5.00, 8.00, 0.20, 20.00},
		{"claude-opus-5", "claude-opus-5", 5.00, 6.25, 10.00, 0.50, 25.00},
		{"claude-opus-4-8", "claude-opus-4-8", 5.00, 6.25, 10.00, 0.50, 25.00},
		{"claude-opus-4-7", "claude-opus-4-7", 5.00, 6.25, 10.00, 0.50, 25.00},
		{"claude-opus-4-6", "claude-opus-4-6", 5.00, 6.25, 10.00, 0.50, 25.00},
		{"claude-opus-4-5-20251101", "claude-opus-4-5", 5.00, 6.25, 10.00, 0.50, 25.00},
		{"claude-opus-4-1-20250805", "claude-opus-4-1", 15.00, 18.75, 30.00, 1.50, 75.00},
		{"claude-opus-4-20250514", "claude-opus-4", 15.00, 18.75, 30.00, 1.50, 75.00},
		{"claude-sonnet-5", "claude-sonnet-5", 2.00, 2.50, 4.00, 0.20, 10.00},
		{"claude-sonnet-4-6", "claude-sonnet-4", 3.00, 3.75, 6.00, 0.30, 15.00},
		{"claude-sonnet-4-5-20250929", "claude-sonnet-4", 3.00, 3.75, 6.00, 0.30, 15.00},
		{"claude-haiku-4-5-20251001", "claude-haiku-4-5", 1.00, 1.25, 2.00, 0.10, 5.00},
		{"claude-haiku-3-5-20241022", "claude-haiku-3-5", 0.80, 1.00, 1.60, 0.08, 4.00},
	}

	for _, c := range cases {
		t.Run(c.model, func(t *testing.T) {
			r, ok := LookupRates(c.model)
			if !ok {
				t.Fatalf("%s: not found in Rates", c.model)
			}
			if r.Family != c.family {
				t.Fatalf("%s: family %q, want %q", c.model, r.Family, c.family)
			}

			check := func(class string, got, want float64) {
				if !approx(got, want) {
					t.Errorf("%s %s: $%.4f for 1M tokens, want $%.4f", c.model, class, got, want)
				}
			}
			check("input", Calculate(c.model, TokenUsage{InputTokens: mtok}).TotalCost, c.input)
			check("output", Calculate(c.model, TokenUsage{OutputTokens: mtok}).TotalCost, c.output)
			check("cache read", Calculate(c.model, TokenUsage{CacheReadTokens: mtok}).TotalCost, c.cacheRead)
			check("cache write 5m", Calculate(c.model, TokenUsage{CacheWriteTokens: mtok}).TotalCost, c.write5m)
			check("cache write 1h", Calculate(c.model, TokenUsage{CacheWriteTokens: mtok, CacheWrite1hTokens: mtok}).TotalCost, c.write1h)
		})
	}
}

// More specific prefixes must win over shorter ones within a family.
func TestPrefixOrdering(t *testing.T) {
	cases := map[string]string{
		"claude-opus-5-5-xxx":       "claude-opus-5-5",
		"claude-opus-5-xxx":         "claude-opus-5",
		"claude-opus-5":             "claude-opus-5",
		"claude-fable-5-1":          "claude-fable-5-1",
		"claude-fable-5":            "claude-fable-5",
		"claude-mythos-5-1-preview": "claude-mythos-5-1",
		"claude-haiku-4-5-20251001": "claude-haiku-4-5",
		"claude-opus-4-8":           "claude-opus-4-8",
		"claude-opus-4-1-20250805":  "claude-opus-4-1",
		"claude-sonnet-5":           "claude-sonnet-5",
		"claude-sonnet-4-6":         "claude-sonnet-4",
	}
	for model, want := range cases {
		r, ok := LookupRates(model)
		if !ok || r.Family != want {
			t.Errorf("%s: family %q (found=%v), want %q", model, r.Family, ok, want)
		}
	}

	// Structural invariant: no entry may be preceded by an entry that is a
	// prefix of it, otherwise the shorter prefix would shadow it.
	for i := range Rates {
		for j := 0; j < i; j++ {
			if strings.HasPrefix(Rates[i].Family, Rates[j].Family) {
				t.Errorf("Rates[%d] %q is shadowed by earlier Rates[%d] %q", i, Rates[i].Family, j, Rates[j].Family)
			}
		}
	}
}

func TestUnknownModelFallback(t *testing.T) {
	ResetUnknownModels()
	defer ResetUnknownModels()

	r, ok := LookupRates("claude-newthing-9")
	if ok {
		t.Fatalf("unknown model reported as known: %+v", r)
	}
	if r.Family != FallbackFamily {
		t.Fatalf("fallback family %q, want %q", r.Family, FallbackFamily)
	}
	if GetRates("claude-newthing-9").Family != FallbackFamily {
		t.Fatal("GetRates did not return the fallback for an unknown model")
	}

	cb := Calculate("claude-newthing-9", TokenUsage{InputTokens: mtok})
	if !approx(cb.TotalCost, Fallback.InputPerMToken) {
		t.Errorf("fallback input cost $%.4f, want $%.4f", cb.TotalCost, Fallback.InputPerMToken)
	}
	Calculate("claude-newthing-9", TokenUsage{OutputTokens: 1})
	Calculate("claude-other-1", TokenUsage{OutputTokens: 1})

	got := UnknownModels()
	if len(got) != 2 || got[0].Model != "claude-newthing-9" || got[0].Count != 2 || got[1].Count != 1 {
		t.Errorf("unknown model counts = %+v, want newthing-9:2, other-1:1", got)
	}

	// Known models must not be counted.
	Calculate("claude-opus-5", TokenUsage{OutputTokens: 1})
	if len(UnknownModels()) != 2 {
		t.Error("a known model was counted as unknown")
	}
}

// Mixed-TTL cache writes: the 1h subset at the 1h rate, the rest at 5m, and an
// oversized 1h figure is clamped to the total.
func TestCacheWriteSplit(t *testing.T) {
	cb := Calculate("claude-opus-5", TokenUsage{CacheWriteTokens: mtok, CacheWrite1hTokens: 250_000})
	want := 0.75*6.25 + 0.25*10.00
	if !approx(cb.CacheWriteCost, want) {
		t.Errorf("split cache write $%.4f, want $%.4f", cb.CacheWriteCost, want)
	}

	cb = Calculate("claude-opus-5", TokenUsage{CacheWriteTokens: mtok, CacheWrite1hTokens: 5 * mtok})
	if !approx(cb.CacheWriteCost, 10.00) {
		t.Errorf("clamped cache write $%.4f, want $10.0000", cb.CacheWriteCost)
	}
}
