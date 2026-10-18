package storage

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/Wather17/higher/internal/reserve"
)

func TestReserveStoreLoadMissingFileReturnsUnconfiguredDocument(t *testing.T) {
	store := newTestReserveStore(t)

	document, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if document.Version != reserve.CurrentDocumentVersion || document.NextID != 1 || document.IsConfigured() || len(document.Entries) != 0 {
		t.Fatalf("unexpected empty reserve document: %+v", document)
	}
}

func TestReserveStoreSaveAndLoadRoundTrip(t *testing.T) {
	store := newTestReserveStore(t)
	document := reserve.NewDocument()
	if err := document.Setup(reserve.SetupInput{IncomeCents: 500000, SaveRateBasisPoints: 1000}); err != nil {
		t.Fatalf("Setup: %v", err)
	}
	if _, err := document.Deposit(reserve.EntryInput{AmountCents: 50000, Date: "2026-09-23", Note: "aporte"}, nowForTest()); err != nil {
		t.Fatalf("Deposit: %v", err)
	}

	if err := store.Save(document); err != nil {
		t.Fatalf("Save: %v", err)
	}
	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.Configuration == nil || len(loaded.Entries) != 1 || loaded.Entries[0].Note != "aporte" {
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

func TestReserveStoreRejectsInvalidJSONWithoutOverwritingFile(t *testing.T) {
	store := newTestReserveStore(t)
	if err := os.MkdirAll(filepath.Dir(store.Path()), 0o700); err != nil {
		t.Fatalf("create test directory: %v", err)
	}
	original := []byte(`{"version": 1`)
	if err := os.WriteFile(store.Path(), original, 0o600); err != nil {
		t.Fatalf("write invalid document: %v", err)
	}

	if _, err := store.Load(); err == nil || !strings.Contains(err.Error(), "decode reserve file") {
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

func TestReserveStoreRejectsInvalidDocument(t *testing.T) {
	store := newTestReserveStore(t)
	document := reserve.NewDocument()
	document.Configuration = &reserve.Configuration{IncomeCents: 500000, SaveRateBasisPoints: 1000, TargetCents: 3000000}
	document.Entries = []reserve.Entry{{ID: 1, Kind: reserve.EntryWithdrawal, AmountCents: 100, Date: "2026-09-23"}}

	if err := store.Save(document); !errors.Is(err, reserve.ErrInvalidDocument) {
		t.Fatalf("expected ErrInvalidDocument, got %v", err)
	}
	if _, err := os.Stat(store.Path()); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("invalid save created a file: %v", err)
	}
}

func newTestReserveStore(t *testing.T) *ReserveStore {
	t.Helper()
	store, err := NewReserveStore(filepath.Join(t.TempDir(), "nested", "reserve.json"))
	if err != nil {
		t.Fatalf("NewReserveStore: %v", err)
	}
	return store
}

func nowForTest() (now time.Time) {
	return time.Date(2026, 9, 23, 0, 0, 0, 0, time.UTC)
}
