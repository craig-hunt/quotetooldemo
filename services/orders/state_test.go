package main

import "testing"

// Order state machine:
// open        -> in_progress | fulfilled | cancelled
// in_progress -> fulfilled | cancelled
// fulfilled   -> (terminal)
// cancelled   -> (terminal)

func TestAllowedTransition_Matrix(t *testing.T) {
	states := []OrderStatus{StatusOpen, StatusInProgress, StatusFulfilled, StatusCancelled}
	allowed := map[OrderStatus]map[OrderStatus]bool{
		StatusOpen: {
			StatusInProgress: true,
			StatusFulfilled:  true,
			StatusCancelled:  true,
		},
		StatusInProgress: {
			StatusFulfilled: true,
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
	unknown := OrderStatus("no_such_status")
	for _, to := range []OrderStatus{StatusOpen, StatusInProgress, StatusFulfilled, StatusCancelled} {
		if allowedTransition(unknown, to) {
			t.Errorf("unknown status must not allow %s", to)
		}
	}
}
