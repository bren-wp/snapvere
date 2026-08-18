package i18n

import "testing"

func TestResolveExactBaseAndFallback(t *testing.T) {
	code, hr := Resolve("hr-HR")
	if code != "hr" || hr["settings"] == "" {
		t.Fatalf("Croatian locale resolution failed: code=%q settings=%q", code, hr["settings"])
	}
	code, en := Resolve("zz-ZZ")
	if code != "en" || en["settings"] == "" {
		t.Fatalf("fallback locale resolution failed: code=%q settings=%q", code, en["settings"])
	}
}

func TestCatalogShape(t *testing.T) {
	loadOnce.Do(load)
	if len(loaded.Languages) < 41 {
		t.Fatalf("expected at least 41 languages, got %d", len(loaded.Languages))
	}
	want := len(loaded.Languages["en"])
	if want < 112 {
		t.Fatalf("expected at least 112 English keys, got %d", want)
	}
	for code, table := range loaded.Languages {
		if len(table) != want {
			t.Fatalf("language %s has %d keys, want %d", code, len(table), want)
		}
	}
}
