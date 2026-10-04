package postgres

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestOpenRejectsInvalidDatabaseURLWithoutExposingIt(t *testing.T) {
	const databaseURL = "invalid-secret-database-url"

	pool, err := Open(context.Background(), databaseURL, time.Second)
	if pool != nil {
		pool.Close()
		t.Fatal("Open() returned a pool for an invalid DATABASE_URL")
	}
	if err == nil {
		t.Fatal("Open() returned no error for an invalid DATABASE_URL")
	}
	if strings.Contains(err.Error(), databaseURL) {
		t.Fatal("Open() exposed DATABASE_URL contents in its error")
	}
}
