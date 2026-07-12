package main

import "testing"

// Quote state machine allowed transitions:
// draft    -> sent
// sent     -> accepted | rejected | expired
// accepted -> (terminal)
// rejected -> (terminal)
// expired  -> (terminal)

func TestAllowedTransition_Matrix(t *testing.T) {
	states := []QuoteStatus{
		StatusDraft, StatusSent, StatusAccepted, StatusRejected, StatusExpired,
	}
	allowed := map[QuoteStatus]map[QuoteStatus]bool{
		StatusDraft: {StatusSent: true},
		StatusSent: {
			StatusAccepted: true,
			StatusRejected: true,
			StatusExpired:  true,
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
	// Guardrail: an unknown status must map to zero transitions.
	unknown := QuoteStatus("no_such_status")
	for _, to := range []QuoteStatus{
		StatusDraft, StatusSent, StatusAccepted, StatusRejected, StatusExpired,
	} {
		if allowedTransition(unknown, to) {
			t.Errorf("unknown status must not allow %s", to)
		}
	}
}
