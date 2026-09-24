package cmd

import (
	"fmt"

	"github.com/ksred/cctrack/internal/calculator"
	"github.com/ksred/cctrack/internal/config"
	"github.com/ksred/cctrack/internal/parser"
	"github.com/ksred/cctrack/internal/store"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Print spend summary to stdout",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		s, err := store.Open(cfg.DBPath)
		if err != nil {
			return fmt.Errorf("opening store: %w", err)
		}
		defer s.Close()

		// Parse latest data
		p := parser.New(s)
		p.ParseAll(cfg.LogDir)

		// Get summary
		summary, err := s.GetSummary()
		if err != nil {
			return fmt.Errorf("getting summary: %w", err)
		}

		// Get top session
		top, err := s.TopSessions(1)
		if err != nil {
			return fmt.Errorf("getting top sessions: %w", err)
		}

		fmt.Printf("cctrack v%s\n", Version)
		fmt.Println("─────────────────────────────────────")
		fmt.Printf("Today          $%-8s (%s tokens)\n", fmtCost(summary.Today.Cost), fmtTokens(summary.Today.Tokens))
		fmt.Printf("This week      $%-8s (%s tokens)\n", fmtCost(summary.Week.Cost), fmtTokens(summary.Week.Tokens))
		fmt.Printf("This month     $%-8s (%s tokens)\n", fmtCost(summary.Month.Cost), fmtTokens(summary.Month.Tokens))
		fmt.Printf("Projected      $%s\n", fmtCost(summary.Projected))
		fmt.Println("─────────────────────────────────────")

		if len(top) > 0 {
			name := top[0].Project
			if name == "" {
				name = top[0].Slug
			}
			if name == "" {
				name = top[0].ID[:8]
			}
			fmt.Printf("Top session: \"%s\" — $%s\n", name, fmtCost(top[0].TotalCost))
		}

		// Report any stored requests whose model matches no pricing entry;
		// those were priced at the fallback rates rather than a list price.
		counts, err := s.GetModelRequestCounts()
		if err != nil {
			return fmt.Errorf("getting model counts: %w", err)
		}
		for _, mc := range counts {
			if _, known := calculator.LookupRates(mc.Model); !known {
				fmt.Printf("Warning: %d requests on unknown model %q priced at fallback (%s)\n",
					mc.Count, mc.Model, calculator.FallbackFamily)
			}
		}

		return nil
	},
}

func fmtCost(v float64) string {
	if v < 0.01 {
		return fmt.Sprintf("%.4f", v)
	}
	if v < 10 {
		return fmt.Sprintf("%.2f", v)
	}
	return fmt.Sprintf("%.2f", v)
}

func fmtTokens(n int64) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	if n < 1_000_000 {
		return fmt.Sprintf("%.1fK", float64(n)/1000)
	}
	return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
