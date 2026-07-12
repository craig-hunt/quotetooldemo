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

	migrationPath := filepath.Join("..", "..", "migrations", "04_invoices.sql")
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

func sampleOrderInfo() OrderInfo {
	return OrderInfo{
		ID:          uuid.New(),
		CustomerID:  uuid.New(),
		Status:      "fulfilled",
		TotalAmount: 5000.00,
	}
}

func TestPGStore_CreateAssignsInvoiceNumber(t *testing.T) {
	ctx := context.Background()
	o := sampleOrderInfo()

	inv, err := testStore.Create(ctx, CreateInput{OrderID: o.ID, DueDays: 30, Notes: "net-30"}, o)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if inv.ID == uuid.Nil {
		t.Fatal("expected id")
	}
	if inv.InvoiceNumber == "" {
		t.Fatal("invoice number blank")
	}
	if inv.AmountDue != o.TotalAmount {
		t.Errorf("amount_due = %v, want %v", inv.AmountDue, o.TotalAmount)
	}
	if inv.Status != StatusDraft {
		t.Errorf("initial status = %s, want %s", inv.Status, StatusDraft)
	}
	if inv.DueDate.Before(time.Now()) {
		t.Errorf("due_date %v not in the future", inv.DueDate)
	}
}

func TestPGStore_GetNotFound(t *testing.T) {
	if _, err := testStore.Get(context.Background(), uuid.New()); !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestPGStore_TransitionMarksPaid(t *testing.T) {
	ctx := context.Background()
	o := sampleOrderInfo()
	inv, err := testStore.Create(ctx, CreateInput{OrderID: o.ID, DueDays: 30}, o)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// draft → sent OK.
	if _, err := testStore.Transition(ctx, inv.ID, StatusSent); err != nil {
		t.Fatalf("draft → sent: %v", err)
	}
	// sent → draft REJECTED.
	if _, err := testStore.Transition(ctx, inv.ID, StatusDraft); !errors.Is(err, ErrInvalidStatus) {
		t.Errorf("sent → draft: want ErrInvalidStatus, got %v", err)
	}
	// sent → paid OK and stamps amount_paid = amount_due.
	got, err := testStore.Transition(ctx, inv.ID, StatusPaid)
	if err != nil {
		t.Fatalf("sent → paid: %v", err)
	}
	if got.Status != StatusPaid {
		t.Errorf("status = %s, want paid", got.Status)
	}
	if got.AmountPaid != got.AmountDue {
		t.Errorf("amount_paid = %v, want %v", got.AmountPaid, got.AmountDue)
	}
	if got.PaidDate == nil {
		t.Errorf("paid_date not stamped")
	}
}

func TestPGStore_ListFiltersByStatus(t *testing.T) {
	ctx := context.Background()
	o := sampleOrderInfo()
	for i := 0; i < 3; i++ {
		if _, err := testStore.Create(ctx, CreateInput{OrderID: o.ID, DueDays: 30}, o); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	draft := StatusDraft
	rows, err := testStore.List(ctx, ListParams{Limit: 10, Offset: 0, Status: &draft})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(rows) < 3 {
		t.Errorf("draft rows = %d, want >= 3", len(rows))
	}
	for _, r := range rows {
		if r.Status != StatusDraft {
			t.Errorf("row status = %s, want draft", r.Status)
		}
	}
}
