package main

import "testing"

// Invoice state machine:
// draft     -> sent | cancelled
// sent      -> paid | overdue | cancelled
// overdue   -> paid | cancelled
// paid      -> (terminal)
// cancelled -> (terminal)

func TestAllowedTransition_Matrix(t *testing.T) {
	states := []InvoiceStatus{
		StatusDraft, StatusSent, StatusOverdue, StatusPaid, StatusCancelled,
	}
	allowed := map[InvoiceStatus]map[InvoiceStatus]bool{
		StatusDraft: {
			StatusSent:      true,
			StatusCancelled: true,
		},
		StatusSent: {
			StatusPaid:      true,
			StatusOverdue:   true,
			StatusCancelled: true,
		},
		StatusOverdue: {
			StatusPaid:      true,
			StatusCancelled: true,
		},
	}
	for _, from := range states {
		for _, to := range states {
			want := allowed[from][to]
			got := allowedTransition(from, to)
			if got != want {
				t.Errorf("allowedTransition(%s, %s) = %v, want %v", from, to, got, want)
			}
		}
	}
}

func TestAllowedTransition_UnknownStateRejectsAll(t *testing.T) {
	unknown := InvoiceStatus("no_such_status")
	for _, to := range []InvoiceStatus{
		StatusDraft, StatusSent, StatusOverdue, StatusPaid, StatusCancelled,
	} {
		if allowedTransition(unknown, to) {
			t.Errorf("unknown status must not allow %s", to)
		}
	}
}
