package main

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func validLineItem() LineItemInput {
	return LineItemInput{
		AreaSqft:        500,
		DepthInches:     2,
		MixType:         MixHMASurface,
		UnitPricePerTon: 95,
	}
}

func validQuoteInput() QuoteInput {
	return QuoteInput{
		CustomerID:  uuid.New(),
		ProjectName: "Main Street repave",
		TaxRate:     0.06,
		MarkupRate:  0.15,
		LineItems:   []LineItemInput{validLineItem()},
	}
}

func TestValidateQuoteInput_Valid(t *testing.T) {
	if err := validateQuoteInput(validQuoteInput()); err != nil {
		t.Fatalf("expected valid input to pass, got %v", err)
	}
	// Each supported mix type must validate.
	for _, mix := range []MixType{MixHMABase, MixHMASurface, MixSuperpave, MixWarmMix} {
		in := validQuoteInput()
		in.LineItems[0].MixType = mix
		if err := validateQuoteInput(in); err != nil {
			t.Errorf("mix %s must be valid, got %v", mix, err)
		}
	}
}

func TestValidateQuoteInput_Rejections(t *testing.T) {
	cases := []struct {
		name     string
		mutate   func(*QuoteInput)
		wantSub  string
	}{
		{
			name:    "missing customer id",
			mutate:  func(q *QuoteInput) { q.CustomerID = uuid.Nil },
			wantSub: ErrMsgCustomerIDRequired,
		},
		{
			name:    "missing project name",
			mutate:  func(q *QuoteInput) { q.ProjectName = "" },
			wantSub: ErrMsgProjectNameRequired,
		},
		{
			name:    "empty line items",
			mutate:  func(q *QuoteInput) { q.LineItems = nil },
			wantSub: ErrMsgLineItemsRequired,
		},
		{
			name:    "zero area",
			mutate:  func(q *QuoteInput) { q.LineItems[0].AreaSqft = 0 },
			wantSub: ErrMsgInvalidArea,
		},
		{
			name:    "negative area",
			mutate:  func(q *QuoteInput) { q.LineItems[0].AreaSqft = -1 },
			wantSub: ErrMsgInvalidArea,
		},
		{
			name:    "zero depth",
			mutate:  func(q *QuoteInput) { q.LineItems[0].DepthInches = 0 },
			wantSub: ErrMsgInvalidDepth,
		},
		{
			name:    "negative depth",
			mutate:  func(q *QuoteInput) { q.LineItems[0].DepthInches = -0.5 },
			wantSub: ErrMsgInvalidDepth,
		},
		{
			name:    "zero unit price",
			mutate:  func(q *QuoteInput) { q.LineItems[0].UnitPricePerTon = 0 },
			wantSub: ErrMsgInvalidUnitPrice,
		},
		{
			name:    "negative unit price",
			mutate:  func(q *QuoteInput) { q.LineItems[0].UnitPricePerTon = -1 },
			wantSub: ErrMsgInvalidUnitPrice,
		},
		{
			name:    "empty mix type",
			mutate:  func(q *QuoteInput) { q.LineItems[0].MixType = "" },
			wantSub: ErrMsgInvalidMixType,
		},
		{
			name:    "unknown mix type",
			mutate:  func(q *QuoteInput) { q.LineItems[0].MixType = "gravel_mix" },
			wantSub: ErrMsgInvalidMixType,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := validQuoteInput()
			tc.mutate(&in)
			err := validateQuoteInput(in)
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tc.wantSub) {
				t.Errorf("error %q missing substring %q", err.Error(), tc.wantSub)
			}
		})
	}
}

func TestValidateQuoteInput_LinePrefixCarriesIndex(t *testing.T) {
	// Second line item invalid must surface index 1, not 0.
	in := validQuoteInput()
	in.LineItems = append(in.LineItems, validLineItem())
	in.LineItems[1].AreaSqft = 0

	err := validateQuoteInput(in)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), ErrMsgLinePrefix+"1") {
		t.Errorf("error %q must reference line 1", err.Error())
	}
}
