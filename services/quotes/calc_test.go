package main

import "testing"

// Table-driven tests are the idiomatic Go pattern. One test function drives
// many cases through a struct slice, making it easy to add new cases.

// Density constant: 12.5 lbs/sqft/inch.
// Formula: tons = (area × depth × 12.5) / 2000

func TestTonsForLine(t *testing.T) {
	cases := []struct {
		name     string
		area     float64
		depth    float64
		wantTons float64
	}{
		{"typical driveway", 500, 2, 6.25},         // 500*2*12.5/2000
		{"parking lot", 20000, 3, 375.0},           // 20000*3*12.5/2000
		{"thin overlay", 1000, 1.5, 9.375},         // 1000*1.5*12.5/2000
		{"zero area", 0, 3, 0.0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tonsForLine(tc.area, tc.depth)
			if got != tc.wantTons {
				t.Errorf("tonsForLine(%v, %v) = %v, want %v",
					tc.area, tc.depth, got, tc.wantTons)
			}
		})
	}
}

func TestLineTotal(t *testing.T) {
	got := lineTotalForLine(100.0, 95.00)
	want := 9500.00
	if got != want {
		t.Errorf("lineTotalForLine(100, 95) = %v, want %v", got, want)
	}
}

func TestComputeQuoteTotals(t *testing.T) {
	q := &Quote{
		TaxRate:    0.06,
		MarkupRate: 0.15,
		LineItems: []LineItem{
			{AreaSqft: 500, DepthInches: 2, UnitPricePerTon: 100},
			{AreaSqft: 1000, DepthInches: 3, UnitPricePerTon: 95},
		},
	}
	computeQuoteTotals(q)

	// Line 1: 500*2*12.5/2000 = 6.25 tons, at $100/ton = $625.00
	// Line 2: 1000*3*12.5/2000 = 18.75 tons, at $95/ton = $1781.25
	// Subtotal: 625 + 1781.25 = 2406.25
	// Tax: 2406.25 * 0.06 = 144.375, rounded to 144.38
	// Markup: (2406.25 + 144.38) * 0.15 = 2550.63 * 0.15 = 382.5945, rounded to 382.59
	// Total: 2406.25 + 144.38 + 382.59 = 2933.22
	if q.LineItems[0].Tons != 6.25 {
		t.Errorf("line 0 tons = %v, want 6.25", q.LineItems[0].Tons)
	}
	if q.LineItems[0].LineTotal != 625.00 {
		t.Errorf("line 0 total = %v, want 625.00", q.LineItems[0].LineTotal)
	}
	if q.LineItems[1].Tons != 18.75 {
		t.Errorf("line 1 tons = %v, want 18.75", q.LineItems[1].Tons)
	}
	if q.LineItems[1].LineTotal != 1781.25 {
		t.Errorf("line 1 total = %v, want 1781.25", q.LineItems[1].LineTotal)
	}
	if q.Subtotal != 2406.25 {
		t.Errorf("subtotal = %v, want 2406.25", q.Subtotal)
	}
	if q.TaxAmount != 144.38 {
		t.Errorf("tax = %v, want 144.38", q.TaxAmount)
	}
	if q.MarkupAmount != 382.59 {
		t.Errorf("markup = %v, want 382.59", q.MarkupAmount)
	}
	if q.Total != 2933.22 {
		t.Errorf("total = %v, want 2933.22", q.Total)
	}
}
