package history

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestAddLoadPruneAndDedupe(t *testing.T) {
	dir := t.TempDir()
	index := filepath.Join(dir, "history.json")
	a := filepath.Join(dir, "a.png")
	b := filepath.Join(dir, "b.avi")
	if err := os.WriteFile(a, []byte("a"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(b, []byte("bb"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := Add(index, Entry{Path: a, Kind: "image", Created: time.Unix(1, 0)}, 10); err != nil {
		t.Fatal(err)
	}
	if err := Add(index, Entry{Path: b, Kind: "video", Created: time.Unix(2, 0)}, 10); err != nil {
		t.Fatal(err)
	}
	if err := Add(index, Entry{Path: a, Kind: "image", Created: time.Unix(3, 0)}, 10); err != nil {
		t.Fatal(err)
	}
	got, err := Load(index, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d entries, want 2", len(got))
	}
	if filepath.Clean(got[0].Path) != filepath.Clean(a) {
		t.Fatalf("newest/deduped entry mismatch: %q", got[0].Path)
	}
	if err := os.Remove(b); err != nil {
		t.Fatal(err)
	}
	got, err = RemoveMissing(index, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d entries after prune, want 1", len(got))
	}
}

func TestLimit(t *testing.T) {
	dir := t.TempDir()
	index := filepath.Join(dir, "history.json")
	for i := 0; i < 4; i++ {
		p := filepath.Join(dir, string(rune('a'+i))+".png")
		if err := os.WriteFile(p, []byte{byte(i)}, 0600); err != nil {
			t.Fatal(err)
		}
		if err := Add(index, Entry{Path: p, Kind: "image", Created: time.Unix(int64(i+1), 0)}, 2); err != nil {
			t.Fatal(err)
		}
	}
	got, err := Load(index, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d entries, want 2", len(got))
	}
}

func TestLoadRejectsUnsupportedSchema(t *testing.T) {
	dir := t.TempDir()
	index := filepath.Join(dir, "history.json")
	if err := os.WriteFile(index, []byte(`{"version":999,"entries":[]}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(index, 10); err == nil {
		t.Fatal("expected unsupported schema error")
	}
}

func TestRemoveMissingPersistsCleanup(t *testing.T) {
	dir := t.TempDir()
	index := filepath.Join(dir, "history.json")
	keep := filepath.Join(dir, "keep.png")
	missing := filepath.Join(dir, "missing.png")
	if err := os.WriteFile(keep, []byte("ok"), 0600); err != nil {
		t.Fatal(err)
	}
	d := document{Version: schemaVersion, Entries: []Entry{
		{Path: missing, Kind: "image", Created: time.Now().Add(time.Minute)},
		{Path: keep, Kind: "image", Created: time.Now()},
	}}
	if err := writeAtomic(index, d); err != nil {
		t.Fatal(err)
	}
	cleaned, err := RemoveMissing(index, 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(cleaned) != 1 || !samePath(cleaned[0].Path, keep) {
		t.Fatalf("unexpected cleaned entries: %#v", cleaned)
	}
	raw, err := os.ReadFile(index)
	if err != nil {
		t.Fatal(err)
	}
	var persisted document
	if err := json.Unmarshal(raw, &persisted); err != nil {
		t.Fatal(err)
	}
	if len(persisted.Entries) != 1 || !samePath(persisted.Entries[0].Path, keep) {
		t.Fatalf("cleanup was not persisted: %#v", persisted.Entries)
	}
}
