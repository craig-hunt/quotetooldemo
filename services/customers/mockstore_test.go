package main

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

var errMockUnexpected = errors.New("mock: no scripted response")

type MockStore struct {
	CreateFn func(ctx context.Context, in CustomerInput) (Customer, error)
	GetFn    func(ctx context.Context, id uuid.UUID) (Customer, error)
	ListFn   func(ctx context.Context, p ListParams) ([]Customer, error)
	UpdateFn func(ctx context.Context, id uuid.UUID, in CustomerInput) (Customer, error)
	DeleteFn func(ctx context.Context, id uuid.UUID) error

	LastCreateInput CustomerInput
	LastListParams  ListParams
	LastGetID       uuid.UUID
	LastUpdateID    uuid.UUID
	LastDeleteID    uuid.UUID
}

func (m *MockStore) Create(ctx context.Context, in CustomerInput) (Customer, error) {
	m.LastCreateInput = in
	if m.CreateFn == nil {
		return Customer{}, errMockUnexpected
	}
	return m.CreateFn(ctx, in)
}

func (m *MockStore) Get(ctx context.Context, id uuid.UUID) (Customer, error) {
	m.LastGetID = id
	if m.GetFn == nil {
		return Customer{}, errMockUnexpected
	}
	return m.GetFn(ctx, id)
}

func (m *MockStore) List(ctx context.Context, p ListParams) ([]Customer, error) {
	m.LastListParams = p
	if m.ListFn == nil {
		return nil, errMockUnexpected
	}
	return m.ListFn(ctx, p)
}

func (m *MockStore) Update(ctx context.Context, id uuid.UUID, in CustomerInput) (Customer, error) {
	m.LastUpdateID = id
	if m.UpdateFn == nil {
		return Customer{}, errMockUnexpected
	}
	return m.UpdateFn(ctx, id, in)
}

func (m *MockStore) Delete(ctx context.Context, id uuid.UUID) error {
	m.LastDeleteID = id
	if m.DeleteFn == nil {
		return errMockUnexpected
	}
	return m.DeleteFn(ctx, id)
}
