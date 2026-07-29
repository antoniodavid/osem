package db

// Read-only-verification probes; live only in the verifier scratch copy.

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func countOpenFDs(t *testing.T, path string) int {
	t.Helper()
	entries, err := os.ReadDir("/proc/self/fd")
	if err != nil {
		t.Skipf("no /proc/self/fd: %v", err)
	}
	n := 0
	for _, e := range entries {
		target, err := filepath.EvalSymlinks(filepath.Join("/proc/self/fd", e.Name()))
		if err == nil && target == path {
			n++
		}
	}
	return n
}

// Probe: DeleteSession must roll back everything when a child delete fails mid-transaction.
func TestProbeDeleteSessionRollsBackOnChildDeleteFailure(t *testing.T) {
	path := createTempDB(t, v2Schema+`
CREATE TRIGGER boom BEFORE DELETE ON session_message BEGIN SELECT RAISE(ABORT, 'boom'); END;`)
	now := int64(1764200000000)
	mustExec(t, path, `INSERT INTO session_v2 (id, project_id, slug, directory, title, time_created, time_updated)
		VALUES ('ses_x', 'p1', 'x', '/tmp', 'Rollback Me', ?, ?)`, now, now)
	mustExec(t, path, `INSERT INTO session_message (id, session_id, time_created, time_updated, data) VALUES ('m1','ses_x',?,?,'{}')`, now, now)
	mustExec(t, path, `INSERT INTO session_part (id, session_id, message_id) VALUES ('p1','ses_x','m1')`)

	client, err := NewClient(path)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	err = client.DeleteSession("ses_x")
	if err == nil {
		t.Fatal("want error from forced child-delete failure, got nil")
	}
	var sessions, messages, parts int
	client.db.QueryRow(`SELECT COUNT(*) FROM session_v2 WHERE id='ses_x'`).Scan(&sessions)
	client.db.QueryRow(`SELECT COUNT(*) FROM session_message`).Scan(&messages)
	client.db.QueryRow(`SELECT COUNT(*) FROM session_part`).Scan(&parts)
	if sessions != 1 || messages != 1 || parts != 1 {
		t.Errorf("rollback incomplete: sessions=%d messages=%d parts=%d, want all 1", sessions, messages, parts)
	}
}

// Probe: a rejected "session not found" delete must leave child rows byte-identical.
func TestProbeSessionNotFoundRollsBackChildDeletes(t *testing.T) {
	path := createTempDB(t, v2Schema)
	now := int64(1764200000000)
	// Orphaned children for a session id that has no session row.
	mustExec(t, path, `INSERT INTO session_message (id, session_id, time_created, time_updated, data) VALUES ('mg','ses_ghost',?,?,'{}')`, now, now)
	mustExec(t, path, `INSERT INTO session_part (id, session_id, message_id) VALUES ('pg','ses_ghost','mg')`)

	client, err := NewClient(path)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	err = client.DeleteSession("ses_ghost")
	if err == nil || !strings.HasPrefix(err.Error(), "session not found: ses_ghost") {
		t.Fatalf("want 'session not found: ses_ghost', got %v", err)
	}
	var messages, parts int
	client.db.QueryRow(`SELECT COUNT(*) FROM session_message`).Scan(&messages)
	client.db.QueryRow(`SELECT COUNT(*) FROM session_part`).Scan(&parts)
	if messages != 1 || parts != 1 {
		t.Errorf("rejected delete mutated children: messages=%d parts=%d, want 1/1", messages, parts)
	}
}

// Probe: the detectSchema error path must close the underlying sql.DB (no fd leak).
func TestProbeNewClientErrorClosesDB(t *testing.T) {
	path := filepath.Join(t.TempDir(), "empty.db")
	d, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.Exec(`CREATE TABLE other (id text PRIMARY KEY)`); err != nil {
		t.Fatal(err)
	}
	d.Close()

	if _, err := NewClient(path); err == nil {
		t.Fatal("want schema-detection error, got nil")
	}
	if n := countOpenFDs(t, path); n != 0 {
		t.Errorf("detected-schema error path leaked %d open fd(s) to %s", n, path)
	}
	_ = fmt.Sprint()
}
