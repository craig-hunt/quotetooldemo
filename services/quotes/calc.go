package main

import "math"

// AsphaltDensityLbsPerSqftInch: pounds per square foot per inch of depth
// for hot-mix asphalt. Derived from ~150 lbs/cubic-foot bulk density:
// 150 lbs/cf ÷ 12 inches/foot = 12.5 lbs/sqft/inch. Industry-common figure.
// Some contractors use 148 lbs/cf (= 12.33 lbs/sqft/inch); the delta lives
// well within the mix-design tolerance for a quote.
const AsphaltDensityLbsPerSqftInch = 12.5

// PoundsPerTon standard conversion.
const PoundsPerTon = 2000.0

// tonsForLine computes the tonnage for one line item.
// tons = (area × depth × density) / 2000
// Rounded to 4 decimal places to match the DB numeric precision.
func tonsForLine(areaSqft, depthInches float64) float64 {
	tons := (areaSqft * depthInches * AsphaltDensityLbsPerSqftInch) / PoundsPerTon
	return round(tons, 4)
}

// lineTotalForLine computes the dollar amount for one line item.
// price = tons × unit_price_per_ton, rounded to cents.
func lineTotalForLine(tons, unitPricePerTon float64) float64 {
	return round(tons*unitPricePerTon, 2)
}

// computeQuoteTotals fills tons and line_total on every line item, then rolls
// up subtotal → tax → markup → total on the quote. Called on create + update.
// Business rule: tax applies to subtotal; markup applies on top of subtotal+tax.
func computeQuoteTotals(q *Quote) {
	var subtotal float64
	for i := range q.LineItems {
		li := &q.LineItems[i]
		li.Tons = tonsForLine(li.AreaSqft, li.DepthInches)
		li.LineTotal = lineTotalForLine(li.Tons, li.UnitPricePerTon)
		subtotal += li.LineTotal
	}
	q.Subtotal = round(subtotal, 2)
	q.TaxAmount = round(q.Subtotal*q.TaxRate, 2)
	q.MarkupAmount = round((q.Subtotal+q.TaxAmount)*q.MarkupRate, 2)
	q.Total = round(q.Subtotal+q.TaxAmount+q.MarkupAmount, 2)
}

// round to n decimal places using standard rounding.
func round(v float64, n int) float64 {
	shift := math.Pow(10, float64(n))
	return math.Round(v*shift) / shift
}
