package main

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

var errMockUnexpected = errors.New("mock: no scripted response")

// MockStore satisfies Store with per-method call injection.
// Set the *Fn field to script the response; leave nil to fail the test.
type MockStore struct {
	CreateFn     func(ctx context.Context, in QuoteInput) (Quote, error)
	GetFn        func(ctx context.Context, id uuid.UUID) (Quote, error)
	ListFn       func(ctx context.Context, p ListParams) ([]Quote, error)
	UpdateFn     func(ctx context.Context, id uuid.UUID, in QuoteInput) (Quote, error)
	TransitionFn func(ctx context.Context, id uuid.UUID, to QuoteStatus) (Quote, error)

	LastListParams  ListParams
	LastTransitTo   QuoteStatus
	LastCreateInput QuoteInput
	LastUpdateID    uuid.UUID
	LastGetID       uuid.UUID
	LastTransitID   uuid.UUID
}

func (m *MockStore) Create(ctx context.Context, in QuoteInput) (Quote, error) {
	m.LastCreateInput = in
	if m.CreateFn == nil {
		return Quote{}, errMockUnexpected
	}
	return m.CreateFn(ctx, in)
}

func (m *MockStore) Get(ctx context.Context, id uuid.UUID) (Quote, error) {
	m.LastGetID = id
	if m.GetFn == nil {
		return Quote{}, errMockUnexpected
	}
	return m.GetFn(ctx, id)
}

func (m *MockStore) List(ctx context.Context, p ListParams) ([]Quote, error) {
	m.LastListParams = p
	if m.ListFn == nil {
		return nil, errMockUnexpected
	}
	return m.ListFn(ctx, p)
}

func (m *MockStore) Update(ctx context.Context, id uuid.UUID, in QuoteInput) (Quote, error) {
	m.LastUpdateID = id
	if m.UpdateFn == nil {
		return Quote{}, errMockUnexpected
	}
	return m.UpdateFn(ctx, id, in)
}

func (m *MockStore) Transition(ctx context.Context, id uuid.UUID, to QuoteStatus) (Quote, error) {
	m.LastTransitID = id
	m.LastTransitTo = to
	if m.TransitionFn == nil {
		return Quote{}, errMockUnexpected
	}
	return m.TransitionFn(ctx, id, to)
}
