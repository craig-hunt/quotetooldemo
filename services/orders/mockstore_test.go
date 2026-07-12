package main

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

var errMockUnexpected = errors.New("mock: no scripted response")

type MockStore struct {
	CreateFn     func(ctx context.Context, in CreateInput, q QuoteInfo) (Order, error)
	GetFn        func(ctx context.Context, id uuid.UUID) (Order, error)
	ListFn       func(ctx context.Context, p ListParams) ([]Order, error)
	UpdateFn     func(ctx context.Context, id uuid.UUID, in UpdateInput) (Order, error)
	TransitionFn func(ctx context.Context, id uuid.UUID, to OrderStatus) (Order, error)

	LastCreateInput CreateInput
	LastCreateQuote QuoteInfo
	LastListParams  ListParams
	LastGetID       uuid.UUID
	LastUpdateID    uuid.UUID
	LastTransitID   uuid.UUID
	LastTransitTo   OrderStatus
}

func (m *MockStore) Create(ctx context.Context, in CreateInput, q QuoteInfo) (Order, error) {
	m.LastCreateInput = in
	m.LastCreateQuote = q
	if m.CreateFn == nil {
		return Order{}, errMockUnexpected
	}
	return m.CreateFn(ctx, in, q)
}

func (m *MockStore) Get(ctx context.Context, id uuid.UUID) (Order, error) {
	m.LastGetID = id
	if m.GetFn == nil {
		return Order{}, errMockUnexpected
	}
	return m.GetFn(ctx, id)
}

func (m *MockStore) List(ctx context.Context, p ListParams) ([]Order, error) {
	m.LastListParams = p
	if m.ListFn == nil {
		return nil, errMockUnexpected
	}
	return m.ListFn(ctx, p)
}

func (m *MockStore) Update(ctx context.Context, id uuid.UUID, in UpdateInput) (Order, error) {
	m.LastUpdateID = id
	if m.UpdateFn == nil {
		return Order{}, errMockUnexpected
	}
	return m.UpdateFn(ctx, id, in)
}

func (m *MockStore) Transition(ctx context.Context, id uuid.UUID, to OrderStatus) (Order, error) {
	m.LastTransitID = id
	m.LastTransitTo = to
	if m.TransitionFn == nil {
		return Order{}, errMockUnexpected
	}
	return m.TransitionFn(ctx, id, to)
}
