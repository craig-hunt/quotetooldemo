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

	migrationPath := filepath.Join("..", "..", "migrations", "01_customers.sql")
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

func sampleCustomerInput(name string) CustomerInput {
	return CustomerInput{
		Name:        name,
		ContactName: "Contact " + name,
		Email:       name + "@example.com",
		Phone:       "555-0100",
		BillingAddress: Address{
			Street: "1 Test Way", City: "Wilmington", State: "DE", Zip: "19801",
		},
	}
}

func TestPGStore_CreateAndGet(t *testing.T) {
	ctx := context.Background()
	in := sampleCustomerInput("Alpha")

	created, err := testStore.Create(ctx, in)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.ID == uuid.Nil {
		t.Fatal("expected id, got nil")
	}
	if created.Name != in.Name {
		t.Errorf("Name = %q, want %q", created.Name, in.Name)
	}
	if created.BillingAddress.City != in.BillingAddress.City {
		t.Errorf("City = %q, want %q", created.BillingAddress.City, in.BillingAddress.City)
	}

	got, err := testStore.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != created.ID {
		t.Errorf("Get id mismatch")
	}
	if got.Email != in.Email {
		t.Errorf("Email round-trip failed: %q", got.Email)
	}
}

func TestPGStore_GetNotFound(t *testing.T) {
	if _, err := testStore.Get(context.Background(), uuid.New()); !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestPGStore_UpdatePersists(t *testing.T) {
	ctx := context.Background()
	created, err := testStore.Create(ctx, sampleCustomerInput("Bravo"))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	next := sampleCustomerInput("Bravo Updated")
	updated, err := testStore.Update(ctx, created.ID, next)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Name != "Bravo Updated" {
		t.Errorf("Name = %q, want %q", updated.Name, "Bravo Updated")
	}

	got, err := testStore.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get after Update: %v", err)
	}
	if got.Name != "Bravo Updated" {
		t.Errorf("Persistence failed: got %q", got.Name)
	}
}

func TestPGStore_UpdateNotFound(t *testing.T) {
	if _, err := testStore.Update(context.Background(), uuid.New(), sampleCustomerInput("ghost")); !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestPGStore_SoftDelete(t *testing.T) {
	ctx := context.Background()
	created, err := testStore.Create(ctx, sampleCustomerInput("Charlie"))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := testStore.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := testStore.Get(ctx, created.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
	if err := testStore.Delete(ctx, created.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("second Delete should return ErrNotFound, got %v", err)
	}
}

func TestPGStore_ListPaginates(t *testing.T) {
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		if _, err := testStore.Create(ctx, sampleCustomerInput("List-"+string(rune('A'+i)))); err != nil {
			t.Fatalf("seed Create: %v", err)
		}
	}
	all, err := testStore.List(ctx, ListParams{Limit: 100, Offset: 0})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(all) == 0 {
		t.Fatal("expected rows, got 0")
	}

	page1, err := testStore.List(ctx, ListParams{Limit: 2, Offset: 0})
	if err != nil {
		t.Fatalf("List page1: %v", err)
	}
	page2, err := testStore.List(ctx, ListParams{Limit: 2, Offset: 2})
	if err != nil {
		t.Fatalf("List page2: %v", err)
	}
	if len(page1) != 2 || len(page2) != 2 {
		t.Errorf("page sizes = %d, %d — want 2, 2", len(page1), len(page2))
	}
	if page1[0].ID == page2[0].ID {
		t.Errorf("pages overlap: %v == %v", page1[0].ID, page2[0].ID)
	}
}
