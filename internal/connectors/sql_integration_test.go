package connectors

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresConnectorReadOnly(t *testing.T) {
	dsn := os.Getenv("TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("TEST_POSTGRES_DSN is not set")
	}
	ctx := context.Background()
	db, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	suffix := make([]byte, 12)
	if _, err = rand.Read(suffix); err != nil {
		t.Fatal(err)
	}
	schema := "connector_test_" + hex.EncodeToString(suffix)
	if _, err = db.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal(err)
	}
	defer db.Exec(ctx, "DROP SCHEMA "+schema+" CASCADE")
	if _, err = db.Exec(ctx, "CREATE TABLE "+schema+".jobs (id text, title text, skills jsonb); INSERT INTO "+schema+".jobs VALUES ('one','플랫폼 엔지니어','[\"Go\",\"Linux\"]')"); err != nil {
		t.Fatal(err)
	}
	config := Config{Type: "postgres", Dataset: "jobs", DSN: dsn, Query: "SELECT id, title, skills FROM " + schema + ".jobs"}
	records, err := Fetch(ctx, config)
	if err != nil || len(records) != 1 {
		t.Fatalf("PostgreSQL fetch: %#v, %v", records, err)
	}
	if records[0]["title"] != "플랫폼 엔지니어" {
		t.Fatal("PostgreSQL text mapping failed")
	}
	skills, ok := records[0]["skills"].([]any)
	if !ok || len(skills) != 2 || skills[0] != "Go" {
		t.Fatal("PostgreSQL JSON mapping failed")
	}
	config.Query = "SELECT current_setting('transaction_read_only') AS readonly, current_setting('statement_timeout') AS timeout"
	records, err = Fetch(ctx, config)
	if err != nil || records[0]["readonly"] != "on" || records[0]["timeout"] != "15s" {
		t.Fatalf("DB session constraints not applied: %#v %v", records, err)
	}
	// The function name itself is permitted by the conservative SQL guard. The
	// transaction must still reject a write performed inside that function.
	_, err = db.Exec(ctx, "CREATE FUNCTION "+schema+".try_mutation() RETURNS int LANGUAGE plpgsql AS $fn$ BEGIN INSERT INTO "+schema+".jobs VALUES ('two','unexpected','[]'); RETURN 1; END $fn$")
	if err != nil {
		t.Fatal(err)
	}
	config.Query = "SELECT " + schema + ".try_mutation()"
	if err := ValidateQuery(config.Query); err != nil {
		t.Fatal("test must reach the database read-only protection")
	}
	if records, err = Fetch(ctx, config); err == nil || records != nil {
		t.Fatal("write hidden inside a SELECT function was accepted")
	}
	if strings.Contains(err.Error(), dsn) || strings.Contains(err.Error(), "try_mutation") {
		t.Fatal("database error exposed source query or DSN")
	}
	var count int
	if err = db.QueryRow(ctx, "SELECT count(*) FROM "+schema+".jobs").Scan(&count); err != nil || count != 1 {
		t.Fatal("read-only connector changed source data")
	}
}
