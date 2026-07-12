package main

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

var errMockUnexpected = errors.New("mock: no scripted response")

type MockStore struct {
	CreateFn     func(ctx context.Context, in CreateInput, o OrderInfo) (Invoice, error)
	GetFn        func(ctx context.Context, id uuid.UUID) (Invoice, error)
	ListFn       func(ctx context.Context, p ListParams) ([]Invoice, error)
	TransitionFn func(ctx context.Context, id uuid.UUID, to InvoiceStatus) (Invoice, error)

	LastCreateInput CreateInput
	LastCreateOrder OrderInfo
	LastListParams  ListParams
	LastGetID       uuid.UUID
	LastTransitID   uuid.UUID
	LastTransitTo   InvoiceStatus
}

func (m *MockStore) Create(ctx context.Context, in CreateInput, o OrderInfo) (Invoice, error) {
	m.LastCreateInput = in
	m.LastCreateOrder = o
	if m.CreateFn == nil {
		return Invoice{}, errMockUnexpected
	}
	return m.CreateFn(ctx, in, o)
}

func (m *MockStore) Get(ctx context.Context, id uuid.UUID) (Invoice, error) {
	m.LastGetID = id
	if m.GetFn == nil {
		return Invoice{}, errMockUnexpected
	}
	return m.GetFn(ctx, id)
}

func (m *MockStore) List(ctx context.Context, p ListParams) ([]Invoice, error) {
	m.LastListParams = p
	if m.ListFn == nil {
		return nil, errMockUnexpected
	}
	return m.ListFn(ctx, p)
}

func (m *MockStore) Transition(ctx context.Context, id uuid.UUID, to InvoiceStatus) (Invoice, error) {
	m.LastTransitID = id
	m.LastTransitTo = to
	if m.TransitionFn == nil {
		return Invoice{}, errMockUnexpected
	}
	return m.TransitionFn(ctx, id, to)
}
