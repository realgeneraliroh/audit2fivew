// Copyright (c) 2026 J
// All rights reserved.
// Released under the GNU Affero General Public License v3.0.

package store_test

import (
	"os"
	"path/filepath"
	"testing"

	"audit2fivew/internal/avc"
	"audit2fivew/internal/store"
)

const sampleLine = `type=AVC msg=audit(1609459200.123:456): avc:  denied  { read write } for  pid=1234 comm="httpd" name="index.html" dev="sda1" ino=78901 scontext=system_u:system_r:httpd_t:s0 tcontext=unconfined_u:object_r:user_home_t:s0 tclass=file permissive=0`

func tmpStore(t *testing.T) *store.Store {
	t.Helper()
	dir := t.TempDir()
	s, err := store.New(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestNewCreatesDB(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "sub", "test.db")
	// Parent directory must exist.
	os.MkdirAll(filepath.Dir(dbPath), 0o755)

	s, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	defer s.Close()

	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Fatal("database file was not created")
	}
}

func TestUpsertAndList(t *testing.T) {
	s := tmpStore(t)

	d, err := avc.Parse(sampleLine)
	if err != nil {
		t.Fatalf("avc.Parse: %v", err)
	}

	if err := s.Upsert(d); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	denials, err := s.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(denials) != 1 {
		t.Fatalf("List returned %d rows, want 1", len(denials))
	}

	got := denials[0]
	if got.Comm != "httpd" {
		t.Errorf("Comm = %q, want httpd", got.Comm)
	}
	if got.Count != 1 {
		t.Errorf("Count = %d, want 1", got.Count)
	}
}

func TestUpsertDeduplicates(t *testing.T) {
	s := tmpStore(t)

	d, _ := avc.Parse(sampleLine)

	// Insert the same denial three times.
	for i := 0; i < 3; i++ {
		if err := s.Upsert(d); err != nil {
			t.Fatalf("Upsert #%d: %v", i, err)
		}
	}

	denials, err := s.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(denials) != 1 {
		t.Fatalf("expected 1 row after dedup, got %d", len(denials))
	}
	if denials[0].Count != 3 {
		t.Errorf("Count = %d, want 3", denials[0].Count)
	}
}

func TestListOrdersByLastSeenDesc(t *testing.T) {
	s := tmpStore(t)

	line1 := `type=AVC msg=audit(1000000000.000:1): avc:  denied  { read } for  pid=1 comm="a" scontext=a:a:a_t:s0 tcontext=a:a:b_t:s0 tclass=file permissive=0`
	line2 := `type=AVC msg=audit(2000000000.000:2): avc:  denied  { write } for  pid=2 comm="b" scontext=a:a:c_t:s0 tcontext=a:a:d_t:s0 tclass=file permissive=0`

	d1, _ := avc.Parse(line1)
	d2, _ := avc.Parse(line2)

	s.Upsert(d1)
	s.Upsert(d2)

	denials, _ := s.List()
	if len(denials) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(denials))
	}
	// d2 has a later timestamp so it should come first.
	if denials[0].Comm != "b" {
		t.Errorf("first row Comm = %q, want 'b' (most recent)", denials[0].Comm)
	}
}

