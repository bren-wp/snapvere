package history

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const schemaVersion = 1

type Entry struct {
	Path    string    `json:"path"`
	Kind    string    `json:"kind"`
	Created time.Time `json:"created"`
	Width   int32     `json:"width,omitempty"`
	Height  int32     `json:"height,omitempty"`
	Size    int64     `json:"size,omitempty"`
}

type document struct {
	Version int     `json:"version"`
	Entries []Entry `json:"entries"`
}

func normalizeLimit(limit int) int {
	if limit < 1 {
		return 100
	}
	if limit > 1000 {
		return 1000
	}
	return limit
}

func readDocument(indexPath string) (document, error) {
	b, err := os.ReadFile(indexPath)
	if errors.Is(err, os.ErrNotExist) {
		return document{Version: schemaVersion}, nil
	}
	if err != nil {
		return document{}, err
	}
	var d document
	if err := json.Unmarshal(b, &d); err != nil {
		return document{}, err
	}
	if d.Version != schemaVersion {
		return document{}, errors.New("unsupported history schema version")
	}
	return d, nil
}

func Load(indexPath string, limit int) ([]Entry, error) {
	d, err := readDocument(indexPath)
	if err != nil {
		return nil, err
	}
	return prune(d.Entries, normalizeLimit(limit)), nil
}

func Add(indexPath string, e Entry, limit int) error {
	limit = normalizeLimit(limit)
	if strings.TrimSpace(e.Path) == "" {
		return errors.New("history entry path is empty")
	}
	abs, err := filepath.Abs(e.Path)
	if err == nil {
		e.Path = abs
	}
	if e.Created.IsZero() {
		e.Created = time.Now()
	}
	if st, err := os.Stat(e.Path); err == nil {
		e.Size = st.Size()
	}

	entries, err := Load(indexPath, limit)
	if err != nil {
		// A damaged history index must never break screenshot saving. Start a
		// fresh index; the media files themselves remain untouched.
		entries = nil
	}
	out := make([]Entry, 0, min(limit, len(entries)+1))
	out = append(out, e)
	for _, old := range entries {
		if samePath(old.Path, e.Path) {
			continue
		}
		out = append(out, old)
		if len(out) >= limit {
			break
		}
	}
	return writeAtomic(indexPath, document{Version: schemaVersion, Entries: out})
}

func RemoveMissing(indexPath string, limit int) ([]Entry, error) {
	d, err := readDocument(indexPath)
	if err != nil {
		return nil, err
	}
	cleaned := prune(d.Entries, normalizeLimit(limit))
	if len(cleaned) != len(d.Entries) {
		if err := writeAtomic(indexPath, document{Version: schemaVersion, Entries: cleaned}); err != nil {
			return cleaned, err
		}
	}
	return cleaned, nil
}

func Clear(indexPath string) error {
	if err := os.Remove(indexPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func prune(entries []Entry, limit int) []Entry {
	seen := make(map[string]struct{}, len(entries))
	out := make([]Entry, 0, min(limit, len(entries)))
	sort.SliceStable(entries, func(i, j int) bool { return entries[i].Created.After(entries[j].Created) })
	for _, e := range entries {
		if strings.TrimSpace(e.Path) == "" {
			continue
		}
		key := pathKey(e.Path)
		if _, ok := seen[key]; ok {
			continue
		}
		if _, err := os.Stat(e.Path); err != nil {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, e)
		if len(out) >= limit {
			break
		}
	}
	return out
}

func samePath(a, b string) bool { return pathKey(a) == pathKey(b) }
func pathKey(p string) string   { return strings.ToLower(filepath.Clean(p)) }

func writeAtomic(path string, d document) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".history-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0600); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
