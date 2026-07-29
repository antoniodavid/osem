package db

import (
	"database/sql"
	"testing"
)

// Positive control for countOpenFDs: an open connection must be detected.
func TestProbeFDDetectionSanity(t *testing.T) {
	path := createTempDB(t, v1Schema)
	d, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	var n int
	if err := d.QueryRow(`SELECT COUNT(*) FROM session`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if got := countOpenFDs(t, path); got < 1 {
		t.Fatalf("fd detection sanity failed: %d open fds with a live connection", got)
	}
}
