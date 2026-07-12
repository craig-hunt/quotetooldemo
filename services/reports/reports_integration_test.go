//go:build integration

package main

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"

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
	testPool    *pgxpool.Pool
	testReports *Reports
)

// TestMain applies EVERY migration because reports reads across all four
// schemas. Order matters — the numeric prefix on each file dictates it.
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

	migrationsDir := filepath.Join("..", "..", "migrations")
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		log.Fatalf("read migrations dir: %v", err)
	}
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".sql" {
			continue
		}
		sqlBytes, err := os.ReadFile(filepath.Join(migrationsDir, e.Name()))
		if err != nil {
			log.Fatalf("read %s: %v", e.Name(), err)
		}
		if _, err := pool.Exec(ctx, string(sqlBytes)); err != nil {
			log.Fatalf("apply %s: %v", e.Name(), err)
		}
	}

	testPool = pool
	testReports = NewReports(pool)

	os.Exit(m.Run())
}

func TestReports_QuoteToCashOnEmptyDB(t *testing.T) {
	ctx := context.Background()
	out, err := testReports.QuoteToCash(ctx)
	if err != nil {
		t.Fatalf("QuoteToCash: %v", err)
	}
	// All six stages return even on an empty DB — the UNION ALL structure
	// emits one row per stage regardless of matching row counts.
	if len(out.Stages) != 6 {
		t.Errorf("stages = %d, want 6", len(out.Stages))
	}
	for _, s := range out.Stages {
		if s.Count != 0 {
			t.Errorf("stage %s count = %d, want 0 on empty DB", s.Stage, s.Count)
		}
	}
}

func TestReports_CycleTimesOnEmptyDB(t *testing.T) {
	ctx := context.Background()
	ct, err := testReports.CycleTimes(ctx)
	if err != nil {
		t.Fatalf("CycleTimes: %v", err)
	}
	// COALESCE ensures a zero, not NULL, on an empty DB.
	if ct.QuoteCreatedToAccepted != 0 {
		t.Errorf("QuoteCreatedToAccepted = %v, want 0", ct.QuoteCreatedToAccepted)
	}
	if ct.OrderFulfilledToPaid != 0 {
		t.Errorf("OrderFulfilledToPaid = %v, want 0", ct.OrderFulfilledToPaid)
	}
}

func TestReports_AgingOnEmptyDB(t *testing.T) {
	ctx := context.Background()
	rows, err := testReports.Aging(ctx)
	if err != nil {
		t.Fatalf("Aging: %v", err)
	}
	// GROUP BY returns no rows when the table is empty.
	if len(rows) != 0 {
		t.Errorf("rows = %d, want 0", len(rows))
	}
}

func TestReports_MixBreakdownOnEmptyDB(t *testing.T) {
	ctx := context.Background()
	rows, err := testReports.MixBreakdown(ctx)
	if err != nil {
		t.Fatalf("MixBreakdown: %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("rows = %d, want 0", len(rows))
	}
}

// TestReports_QuoteToCashWithData exercises the reader path against seeded
// rows. Uses direct SQL inserts to avoid depending on other services' code.
func TestReports_QuoteToCashWithData(t *testing.T) {
	ctx := context.Background()
	// Two draft quotes, one sent, zero accepted — so first three stage
	// counts should be 2, 1, 0.
	custID := "11111111-1111-1111-1111-111111111111"
	_, err := testPool.Exec(ctx, `
		INSERT INTO customers.customers (id, name, billing_address)
		VALUES ($1, 'Test Co', '{}'::jsonb)
	`, custID)
	if err != nil {
		t.Fatalf("seed customer: %v", err)
	}
	_, err = testPool.Exec(ctx, `
		INSERT INTO quotes.quotes (customer_id, project_name, project_address, status, subtotal, tax_rate, tax_amount, markup_rate, markup_amount, total)
		VALUES ($1, 'A', '', 'draft', 100, 0.06, 6, 0.15, 15.9, 121.9),
		       ($1, 'B', '', 'draft', 200, 0.06, 12, 0.15, 31.8, 243.8),
		       ($1, 'C', '', 'sent', 300, 0.06, 18, 0.15, 47.7, 365.7)
	`, custID)
	if err != nil {
		t.Fatalf("seed quotes: %v", err)
	}
	_, err = testPool.Exec(ctx, `
		UPDATE quotes.quotes SET sent_at = NOW() WHERE status = 'sent'
	`)
	if err != nil {
		t.Fatalf("stamp sent_at: %v", err)
	}

	out, err := testReports.QuoteToCash(ctx)
	if err != nil {
		t.Fatalf("QuoteToCash: %v", err)
	}
	counts := map[string]int64{}
	for _, s := range out.Stages {
		counts[s.Stage] = s.Count
	}
	if counts["quotes_created"] != 3 {
		t.Errorf("quotes_created = %d, want 3", counts["quotes_created"])
	}
	if counts["quotes_sent"] != 1 {
		t.Errorf("quotes_sent = %d, want 1", counts["quotes_sent"])
	}
	if counts["quotes_accepted"] != 0 {
		t.Errorf("quotes_accepted = %d, want 0", counts["quotes_accepted"])
	}
}
