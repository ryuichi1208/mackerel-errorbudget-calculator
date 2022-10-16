package lib

import (
	mp "github.com/mackerelio/go-mackerel-plugin"
)

type Budget struct {
	Prefix         string
	Name           string
	ViolationCount float64
	ErrorBudget    float64
	BudgetSize     float64
}

func graphGen(labelPrefix, metrics string) map[string]mp.Graphs {
	return map[string]mp.Graphs{
		"violationCount": {
			Label: labelPrefix,
			Unit:  mp.UnitInteger,
			Metrics: []mp.Metrics{
				{Name: "ViolationCount", Label: "violation_count", Diff: false},
				{Name: "ErrorBudget", Label: "error_budget", Diff: false},
			},
		},
		"errorBudget": {
			Label: labelPrefix,
			Unit:  mp.UnitPercentage,
			Metrics: []mp.Metrics{
				{Name: "ErrorBudgetPercent", Label: "budget", Diff: false},
			},
		},
	}
}

func NewBudget(violationCount, errorBudget, budgeSize float64) *Budget {
	return &Budget{
		Prefix:         opts.Prefix,
		ViolationCount: violationCount,
		ErrorBudget:    errorBudget,
		BudgetSize:     budgeSize,
	}
}

func (b Budget) GraphDefinition() map[string]mp.Graphs {
	return graphGen(opts.Prefix, opts.Metrics)
}

func (b Budget) FetchMetrics() (map[string]float64, error) {
	m := make(map[string]float64)

	m["ViolationCount"] = b.ViolationCount
	m["ErrorBudgetPercent"] = b.ErrorBudget
	m["ErrorBudget"] = b.BudgetSize

	return m, nil
}
