//go:build integration

package main

import (
	"context"
	"errors"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	testDBName         = "test"
	testDBUser         = "test"
	testDBPassword     = "test"
	testStartupTimeout = 60 * time.Second
)

var (
	testPool  *pgxpool.Pool
	testStore *PGStore
)

func TestMain(m *testing.M) {
	ctx := context.Background()

	pgContainer, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase(testDBName),
		postgres.WithUsername(testDBUser),
		postgres.WithPassword(testDBPassword),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(testStartupTimeout),
		),
	)
	if err != nil {
		log.Fatalf("run container: %v", err)
	}
	defer func() { _ = pgContainer.Terminate(ctx) }()

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		log.Fatalf("conn string: %v", err)
	}
	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		log.Fatalf("pool: %v", err)
	}
	defer pool.Close()

	migrationPath := filepath.Join("..", "..", "migrations", "03_orders.sql")
	sqlBytes, err := os.ReadFile(migrationPath)
	if err != nil {
		log.Fatalf("read migration: %v", err)
	}
	if _, err := pool.Exec(ctx, string(sqlBytes)); err != nil {
		log.Fatalf("apply migration: %v", err)
	}

	testPool = pool
	testStore = NewPGStore(pool)

	os.Exit(m.Run())
}

func sampleQuoteInfo() QuoteInfo {
	return QuoteInfo{
		ID:         uuid.New(),
		CustomerID: uuid.New(),
		Status:     "accepted",
		Total:      2933.22,
	}
}

func TestPGStore_CreateAssignsOrderNumber(t *testing.T) {
	ctx := context.Background()
	q := sampleQuoteInfo()

	o, err := testStore.Create(ctx, CreateInput{QuoteID: q.ID, Notes: "n/a"}, q)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if o.ID == uuid.Nil {
		t.Fatal("expected id")
	}
	if o.OrderNumber == "" {
		t.Fatal("order number blank")
	}
	if o.TotalAmount != q.Total {
		t.Errorf("total = %v, want %v", o.TotalAmount, q.Total)
	}
	if o.Status != StatusOpen {
		t.Errorf("initial status = %s, want %s", o.Status, StatusOpen)
	}
}

func TestPGStore_GetNotFound(t *testing.T) {
	if _, err := testStore.Get(context.Background(), uuid.New()); !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestPGStore_TransitionEnforcesStateMachine(t *testing.T) {
	ctx := context.Background()
	q := sampleQuoteInfo()
	o, err := testStore.Create(ctx, CreateInput{QuoteID: q.ID}, q)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// open → fulfilled OK (skip-ahead permitted per state machine).
	got, err := testStore.Transition(ctx, o.ID, StatusFulfilled)
	if err != nil {
		t.Fatalf("open → fulfilled: %v", err)
	}
	if got.Status != StatusFulfilled {
		t.Errorf("status = %s, want fulfilled", got.Status)
	}
	if got.FulfilledDate == nil {
		t.Errorf("fulfilled_date not stamped")
	}

	// fulfilled → cancelled REJECTED (terminal).
	if _, err := testStore.Transition(ctx, o.ID, StatusCancelled); !errors.Is(err, ErrInvalidStatus) {
		t.Errorf("fulfilled → cancelled: want ErrInvalidStatus, got %v", err)
	}
}

func TestPGStore_UpdateSchedulesDate(t *testing.T) {
	ctx := context.Background()
	q := sampleQuoteInfo()
	o, err := testStore.Create(ctx, CreateInput{QuoteID: q.ID}, q)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	scheduled := time.Now().Add(48 * time.Hour).UTC()
	updated, err := testStore.Update(ctx, o.ID, UpdateInput{
		ScheduledDate: &scheduled,
		Notes:         "next week",
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.ScheduledDate == nil {
		t.Fatal("scheduled_date nil")
	}
	if updated.Notes != "next week" {
		t.Errorf("notes = %q, want %q", updated.Notes, "next week")
	}
}
