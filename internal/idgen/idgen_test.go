package idgen

import (
	"database/sql"
	"os"
	"sync"
	"testing"

	_ "github.com/go-sql-driver/mysql"
)

func newSeqDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("GANG_TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("GANG_TEST_MYSQL_DSN is required for MySQL-backed idgen tests")
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS id_sequences (name VARCHAR(64) PRIMARY KEY NOT NULL, next_value BIGINT NOT NULL)`); err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := db.Exec(`DELETE FROM id_sequences`); err != nil {
		t.Fatalf("clear: %v", err)
	}
	return db
}

func TestNextSeqMonotonicStart(t *testing.T) {
	db := newSeqDB(t)
	got := NextUserUID(db)
	if got != "10000000" {
		t.Fatalf("first uid = %q, want 10000000", got)
	}
	if got := NextUserUID(db); got != "10000001" {
		t.Fatalf("second uid = %q, want 10000001", got)
	}
	// Independent sequence.
	if got := NextRoomRID(db); got != "20000000" {
		t.Fatalf("first rid = %q, want 20000000", got)
	}
}

func TestNextSeqNoCollisionUnderConcurrency(t *testing.T) {
	db := newSeqDB(t)

	const n = 500
	var (
		mu   sync.Mutex
		seen = make(map[string]struct{}, n)
		wg   sync.WaitGroup
	)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			id := NextUserUID(db)
			mu.Lock()
			seen[id] = struct{}{}
			mu.Unlock()
		}()
	}
	wg.Wait()
	if len(seen) != n {
		t.Fatalf("got %d distinct ids, want %d (collision under concurrency)", len(seen), n)
	}
}
