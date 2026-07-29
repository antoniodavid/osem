package db

import (
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

const v2Schema = `
CREATE TABLE project (id text PRIMARY KEY, name text);
CREATE TABLE session_v2 (
	id text PRIMARY KEY, project_id text NOT NULL, parent_id text, slug text NOT NULL,
	directory text NOT NULL, title text NOT NULL, time_created integer NOT NULL,
	time_updated integer NOT NULL, agent text, model text, cost real DEFAULT 0
);
CREATE TABLE session_message (
	id text PRIMARY KEY, session_id text NOT NULL, time_created integer NOT NULL,
	time_updated integer NOT NULL, data text NOT NULL
);
CREATE TABLE session_part (
	id text PRIMARY KEY, session_id text NOT NULL, message_id text NOT NULL
);
`

const v1Schema = `
CREATE TABLE project (id text PRIMARY KEY, name text);
CREATE TABLE session (
	id text PRIMARY KEY, project_id text NOT NULL, parent_id text, slug text NOT NULL,
	directory text NOT NULL, title text NOT NULL, time_created integer NOT NULL,
	time_updated integer NOT NULL, agent text, model text, cost real DEFAULT 0
);
CREATE TABLE message (
	id text PRIMARY KEY, session_id text NOT NULL, time_created integer NOT NULL,
	time_updated integer NOT NULL, data text NOT NULL
);
`

func createTempDB(t *testing.T, schema string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "opencode.db")
	d, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open temp db: %v", err)
	}
	defer d.Close()
	if _, err := d.Exec(schema); err != nil {
		t.Fatalf("apply schema: %v", err)
	}
	return path
}

func mustExec(t *testing.T, path string, query string, args ...any) {
	t.Helper()
	d, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open temp db: %v", err)
	}
	defer d.Close()
	if _, err := d.Exec(query, args...); err != nil {
		t.Fatalf("exec %q: %v", query, err)
	}
}

func TestListSessionsReadsV2Storage(t *testing.T) {
	path := createTempDB(t, v2Schema)
	now := time.Now().UnixMilli()

	mustExec(t, path, `INSERT INTO project (id, name) VALUES ('p1', 'osem-project')`)
	mustExec(t, path, `INSERT INTO session_v2 (id, project_id, parent_id, slug, directory, title, time_created, time_updated)
		VALUES ('ses_main', 'p1', NULL, 'main', '/tmp/proj', 'V2 Session', ?, ?)`, now, now)
	mustExec(t, path, `INSERT INTO session_v2 (id, project_id, parent_id, slug, directory, title, time_created, time_updated)
		VALUES ('ses_sub', 'p1', 'ses_main', 'sub', '/tmp/proj', 'Subagent Session', ?, ?)`, now, now)
	mustExec(t, path, `INSERT INTO session_message (id, session_id, time_created, time_updated, data) VALUES ('m1', 'ses_main', ?, ?, '{}')`, now, now)
	mustExec(t, path, `INSERT INTO session_message (id, session_id, time_created, time_updated, data) VALUES ('m2', 'ses_main', ?, ?, '{}')`, now, now)

	client, err := NewClient(path)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	sessions, err := client.ListSessions(10)
	if err != nil {
		t.Fatalf("ListSessions: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("want 1 main session, got %d", len(sessions))
	}
	s := sessions[0]
	if s.ID != "ses_main" || s.Title != "V2 Session" {
		t.Errorf("unexpected session: %s %q", s.ID, s.Title)
	}
	if s.ProjectName != "osem-project" {
		t.Errorf("project name = %q, want %q", s.ProjectName, "osem-project")
	}
	if s.MessageCount != 2 {
		t.Errorf("message count = %d, want 2", s.MessageCount)
	}
}

func TestDeleteSessionCleansV2Children(t *testing.T) {
	path := createTempDB(t, v2Schema)
	now := time.Now().UnixMilli()

	mustExec(t, path, `INSERT INTO session_v2 (id, project_id, slug, directory, title, time_created, time_updated)
		VALUES ('ses_del', 'p1', 'del', '/tmp', 'Delete Me', ?, ?)`, now, now)
	mustExec(t, path, `INSERT INTO session_v2 (id, project_id, slug, directory, title, time_created, time_updated)
		VALUES ('ses_keep', 'p1', 'keep', '/tmp', 'Keep Me', ?, ?)`, now, now)
	mustExec(t, path, `INSERT INTO session_message (id, session_id, time_created, time_updated, data)
		VALUES ('m_del', 'ses_del', ?, ?, '{}')`, now, now)
	mustExec(t, path, `INSERT INTO session_message (id, session_id, time_created, time_updated, data)
		VALUES ('m_keep', 'ses_keep', ?, ?, '{}')`, now, now)
	mustExec(t, path, `INSERT INTO session_part (id, session_id, message_id) VALUES ('part_del', 'ses_del', 'm_del')`)

	client, err := NewClient(path)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	if err := client.DeleteSession("ses_del"); err != nil {
		t.Fatalf("DeleteSession: %v", err)
	}

	var sessions, messages, parts int
	if err := client.db.QueryRow(`SELECT COUNT(*) FROM session_v2`).Scan(&sessions); err != nil {
		t.Fatal(err)
	}
	if err := client.db.QueryRow(`SELECT COUNT(*) FROM session_message`).Scan(&messages); err != nil {
		t.Fatal(err)
	}
	if err := client.db.QueryRow(`SELECT COUNT(*) FROM session_part`).Scan(&parts); err != nil {
		t.Fatal(err)
	}
	if sessions != 1 || messages != 1 || parts != 0 {
		t.Errorf("after delete: sessions=%d (want 1), messages=%d (want 1), parts=%d (want 0)", sessions, messages, parts)
	}

	if err := client.DeleteSession("ses_missing"); err == nil {
		t.Error("DeleteSession of missing id: want error, got nil")
	}
}

func TestListSessionsFallsBackToV1Storage(t *testing.T) {
	path := createTempDB(t, v1Schema)
	now := time.Now().UnixMilli()

	mustExec(t, path, `INSERT INTO project (id, name) VALUES ('p1', 'legacy-project')`)
	mustExec(t, path, `INSERT INTO session (id, project_id, parent_id, slug, directory, title, time_created, time_updated)
		VALUES ('ses_old', 'p1', NULL, 'old', '/tmp/proj', 'Legacy Session', ?, ?)`, now, now)
	mustExec(t, path, `INSERT INTO message (id, session_id, time_created, time_updated, data) VALUES ('m1', 'ses_old', ?, ?, '{}')`, now, now)

	client, err := NewClient(path)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	sessions, err := client.ListSessions(10)
	if err != nil {
		t.Fatalf("ListSessions: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("want 1 session, got %d", len(sessions))
	}
	if sessions[0].Title != "Legacy Session" || sessions[0].MessageCount != 1 {
		t.Errorf("unexpected session: %+v", sessions[0])
	}
}

func TestNewClientRejectsEmptySchema(t *testing.T) {
	path := createTempDB(t, `CREATE TABLE other (id text PRIMARY KEY)`)
	if _, err := NewClient(path); err == nil {
		t.Fatal("NewClient on db without session tables: want error, got nil")
	}
}
