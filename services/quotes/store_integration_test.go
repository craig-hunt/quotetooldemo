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

	migrationPath := filepath.Join("..", "..", "migrations", "02_quotes.sql")
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

func sampleQuoteInput() QuoteInput {
	return QuoteInput{
		CustomerID:     uuid.New(),
		ProjectName:    "Warehouse Repave",
		ProjectAddress: "1 Test Way",
		TaxRate:        0.06,
		MarkupRate:     0.15,
		LineItems: []LineItemInput{
			{AreaSqft: 500, DepthInches: 2, MixType: MixHMASurface, UnitPricePerTon: 100},
			{AreaSqft: 1000, DepthInches: 3, MixType: MixHMABase, UnitPricePerTon: 88},
		},
	}
}

func TestPGStore_CreateComputesTotals(t *testing.T) {
	ctx := context.Background()
	in := sampleQuoteInput()

	created, err := testStore.Create(ctx, in)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.ID == uuid.Nil {
		t.Fatal("expected id")
	}
	// Server-side tons computed via 12.5 lbs/sqft/inch density.
	// Line 1: 500*2*12.5/2000 = 6.25 tons @ 100 = $625
	// Line 2: 1000*3*12.5/2000 = 18.75 tons @ 88 = $1650
	if len(created.LineItems) != 2 {
		t.Fatalf("line items = %d, want 2", len(created.LineItems))
	}
	if created.LineItems[0].Tons != 6.25 {
		t.Errorf("line 0 tons = %v, want 6.25", created.LineItems[0].Tons)
	}
	if created.Subtotal <= 0 {
		t.Errorf("subtotal not computed: %v", created.Subtotal)
	}
	if created.Total < created.Subtotal {
		t.Errorf("total %v < subtotal %v", created.Total, created.Subtotal)
	}
}

func TestPGStore_GetHydratesLineItems(t *testing.T) {
	ctx := context.Background()
	created, err := testStore.Create(ctx, sampleQuoteInput())
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := testStore.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(got.LineItems) != len(created.LineItems) {
		t.Errorf("line items = %d, want %d", len(got.LineItems), len(created.LineItems))
	}
	if got.Total != created.Total {
		t.Errorf("total round-trip failed: got %v want %v", got.Total, created.Total)
	}
}

func TestPGStore_GetNotFound(t *testing.T) {
	if _, err := testStore.Get(context.Background(), uuid.New()); !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestPGStore_TransitionEnforcesStateMachine(t *testing.T) {
	ctx := context.Background()
	created, err := testStore.Create(ctx, sampleQuoteInput())
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// draft → sent OK.
	if _, err := testStore.Transition(ctx, created.ID, StatusSent); err != nil {
		t.Fatalf("draft → sent: %v", err)
	}
	// sent → draft REJECTED.
	if _, err := testStore.Transition(ctx, created.ID, StatusDraft); !errors.Is(err, ErrInvalidStatus) {
		t.Errorf("sent → draft: want ErrInvalidStatus, got %v", err)
	}
	// sent → accepted OK.
	got, err := testStore.Transition(ctx, created.ID, StatusAccepted)
	if err != nil {
		t.Fatalf("sent → accepted: %v", err)
	}
	if got.Status != StatusAccepted {
		t.Errorf("status = %s, want %s", got.Status, StatusAccepted)
	}
	if got.AcceptedAt == nil {
		t.Errorf("accepted_at not stamped")
	}
	// accepted → anything REJECTED (terminal).
	if _, err := testStore.Transition(ctx, created.ID, StatusSent); !errors.Is(err, ErrInvalidStatus) {
		t.Errorf("accepted → sent: want ErrInvalidStatus, got %v", err)
	}
}

func TestPGStore_UpdateRebuildsLineItems(t *testing.T) {
	ctx := context.Background()
	created, err := testStore.Create(ctx, sampleQuoteInput())
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Update with a single line item.
	next := sampleQuoteInput()
	next.CustomerID = created.CustomerID
	next.LineItems = []LineItemInput{
		{AreaSqft: 100, DepthInches: 1, MixType: MixHMASurface, UnitPricePerTon: 200},
	}
	updated, err := testStore.Update(ctx, created.ID, next)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if len(updated.LineItems) != 1 {
		t.Errorf("line items after update = %d, want 1", len(updated.LineItems))
	}
}
