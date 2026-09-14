package storage

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Wather17/higher/internal/subscription"
)

func TestStoreLoadMissingFileReturnsEmptyDocument(t *testing.T) {
	store := newTestStore(t)

	document, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if document.Version != subscription.CurrentDocumentVersion || document.NextID != 1 || len(document.Subscriptions) != 0 {
		t.Fatalf("unexpected empty document: %+v", document)
	}
}

func TestStoreSaveAndLoadRoundTrip(t *testing.T) {
	store := newTestStore(t)
	document := subscription.NewDocument()
	if _, err := document.Add(subscription.AddInput{Name: "Music", AmountCents: 1090, Period: subscription.PeriodMonthly}); err != nil {
		t.Fatalf("Add: %v", err)
	}

	if err := store.Save(document); err != nil {
		t.Fatalf("Save: %v", err)
	}
	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(loaded.Subscriptions) != 1 || loaded.Subscriptions[0].Name != "Music" {
		t.Fatalf("unexpected loaded document: %+v", loaded)
	}

	if runtime.GOOS != "windows" {
		info, err := os.Stat(store.Path())
		if err != nil {
			t.Fatalf("stat store: %v", err)
		}
		if info.Mode().Perm() != 0o600 {
			t.Fatalf("expected private file permissions, got %o", info.Mode().Perm())
		}
	}
}

func TestStoreRejectsInvalidJSONWithoutOverwritingFile(t *testing.T) {
	store := newTestStore(t)
	if err := os.MkdirAll(filepath.Dir(store.Path()), 0o700); err != nil {
		t.Fatalf("create test directory: %v", err)
	}
	original := []byte(`{"version": 1`)
	if err := os.WriteFile(store.Path(), original, 0o600); err != nil {
		t.Fatalf("write invalid document: %v", err)
	}

	if _, err := store.Load(); err == nil || !strings.Contains(err.Error(), "decode subscriptions file") {
		t.Fatalf("expected decode error, got %v", err)
	}
	content, err := os.ReadFile(store.Path())
	if err != nil {
		t.Fatalf("read invalid document: %v", err)
	}
	if string(content) != string(original) {
		t.Fatalf("invalid document was changed: %q", content)
	}
}

func newTestStore(t *testing.T) *Store {
	t.Helper()
	store, err := NewStore(filepath.Join(t.TempDir(), "nested", "subscriptions.json"))
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	return store
}
