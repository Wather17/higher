package cli

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Wather17/higher/internal/reserve"
	"github.com/Wather17/higher/internal/storage"
)

func TestReserveCommandsSetupDepositWithdrawStatusAndList(t *testing.T) {
	reserveStore, err := storage.NewReserveStore(filepath.Join(t.TempDir(), "reserve.json"))
	if err != nil {
		t.Fatalf("NewReserveStore: %v", err)
	}
	root := NewRootCommand(Dependencies{
		ReserveStore: reserveStore,
		Now:          func() time.Time { return time.Date(2026, 9, 23, 0, 0, 0, 0, time.UTC) },
	})

	if output, err := executeRoot(root, "reserve", "setup", "--income", "5000.00", "--save-rate", "10", "--target", "1000.00"); err != nil {
		t.Fatalf("setup: %v; output: %s", err, output)
	}
	if output, err := executeRoot(root, "reserve", "deposit", "--amount", "500.00", "--note", "aporte mensal"); err != nil {
		t.Fatalf("deposit: %v; output: %s", err, output)
	}
	if output, err := executeRoot(root, "reserve", "withdraw", "--amount", "100.00", "--date", "2026-09-24", "--note", "emergência"); err != nil {
		t.Fatalf("withdraw: %v; output: %s", err, output)
	}

	output, err := executeRoot(root, "reserve", "status")
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	for _, expected := range []string{
		"Renda líquida mensal: BRL 5000.00",
		"Percentual para guardar: 10.00%",
		"Aporte mensal planejado: BRL 500.00",
		"Meta: BRL 1000.00",
		"Saldo: BRL 400.00",
		"Restante: BRL 600.00",
		"Progresso: 40.00%",
		"Previsão: 2 meses",
	} {
		if !strings.Contains(output, expected) {
			t.Errorf("status output does not contain %q:\n%s", expected, output)
		}
	}

	output, err = executeRoot(root, "reserve", "list")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if strings.Index(output, "emergência") > strings.Index(output, "aporte mensal") {
		t.Fatalf("expected newest entry first, got:\n%s", output)
	}
	for _, expected := range []string{"Retirada", "BRL 100.00", "2026-09-24", "Depósito", "BRL 500.00", "aporte mensal"} {
		if !strings.Contains(output, expected) {
			t.Errorf("list output does not contain %q:\n%s", expected, output)
		}
	}
}

func TestReserveCommandsUseDefaultTargetAndRejectDuplicateSetup(t *testing.T) {
	reserveStore, err := storage.NewReserveStore(filepath.Join(t.TempDir(), "reserve.json"))
	if err != nil {
		t.Fatalf("NewReserveStore: %v", err)
	}
	root := NewRootCommand(Dependencies{ReserveStore: reserveStore})

	if _, err := executeRoot(root, "reserve", "setup", "--income", "5000.00", "--save-rate", "10"); err != nil {
		t.Fatalf("setup: %v", err)
	}
	output, err := executeRoot(root, "reserve", "status")
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if !strings.Contains(output, "Meta: BRL 30000.00") {
		t.Fatalf("expected six-income target, got:\n%s", output)
	}

	if _, err := executeRoot(root, "reserve", "deposit", "--amount", "100.00"); err != nil {
		t.Fatalf("deposit: %v", err)
	}
	if _, err := executeRoot(root, "reserve", "setup", "--income", "6000.00", "--save-rate", "20"); !errors.Is(err, reserve.ErrAlreadyConfigured) {
		t.Fatalf("duplicate setup error = %v, expected ErrAlreadyConfigured", err)
	}
	output, err = executeRoot(root, "reserve", "status")
	if err != nil {
		t.Fatalf("status after duplicate setup: %v", err)
	}
	if !strings.Contains(output, "Saldo: BRL 100.00") {
		t.Fatalf("duplicate setup changed balance, got:\n%s", output)
	}
}

func TestReserveCommandsEditPreservesEntries(t *testing.T) {
	reserveStore, err := storage.NewReserveStore(filepath.Join(t.TempDir(), "reserve.json"))
	if err != nil {
		t.Fatalf("NewReserveStore: %v", err)
	}
	root := NewRootCommand(Dependencies{ReserveStore: reserveStore})

	if _, err := executeRoot(root, "reserve", "setup", "--income", "5000.00", "--save-rate", "10"); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if _, err := executeRoot(root, "reserve", "deposit", "--amount", "100.00"); err != nil {
		t.Fatalf("deposit: %v", err)
	}
	if _, err := executeRoot(root, "reserve", "edit", "--income", "6000.00", "--save-rate", "15"); err != nil {
		t.Fatalf("edit: %v", err)
	}
	if _, err := executeRoot(root, "reserve", "edit"); !errors.Is(err, reserve.ErrNoChanges) {
		t.Fatalf("empty edit error = %v, expected ErrNoChanges", err)
	}

	output, err := executeRoot(root, "reserve", "status")
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	for _, expected := range []string{"Renda líquida mensal: BRL 6000.00", "Percentual para guardar: 15.00%", "Saldo: BRL 100.00"} {
		if !strings.Contains(output, expected) {
			t.Errorf("status output does not contain %q:\n%s", expected, output)
		}
	}
}

func TestReserveCommandsRejectUnconfiguredAndInsufficientWithdrawal(t *testing.T) {
	reserveStore, err := storage.NewReserveStore(filepath.Join(t.TempDir(), "reserve.json"))
	if err != nil {
		t.Fatalf("NewReserveStore: %v", err)
	}
	root := NewRootCommand(Dependencies{ReserveStore: reserveStore})

	if _, err := executeRoot(root, "reserve", "status"); !errors.Is(err, reserve.ErrNotConfigured) {
		t.Fatalf("status error = %v, expected ErrNotConfigured", err)
	}
	if _, err := executeRoot(root, "reserve", "setup", "--income", "5000.00", "--save-rate", "10"); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if _, err := executeRoot(root, "reserve", "withdraw", "--amount", "1.00"); !errors.Is(err, reserve.ErrInsufficientBalance) {
		t.Fatalf("withdraw error = %v, expected ErrInsufficientBalance", err)
	}

	document, err := reserveStore.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(document.Entries) != 0 {
		t.Fatalf("failed withdrawal changed entries: %+v", document.Entries)
	}
}

func TestReserveCommandsKeepSubscriptionsInSeparateStore(t *testing.T) {
	directory := t.TempDir()
	subscriptionStore, err := storage.NewStore(filepath.Join(directory, "subscriptions.json"))
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	reserveStore, err := storage.NewReserveStore(filepath.Join(directory, "reserve.json"))
	if err != nil {
		t.Fatalf("NewReserveStore: %v", err)
	}
	root := NewRootCommand(Dependencies{Store: subscriptionStore, ReserveStore: reserveStore})

	if _, err := executeRoot(root, "subscription", "add", "--name", "Music", "--amount", "10.00", "--period", "monthly"); err != nil {
		t.Fatalf("subscription add: %v", err)
	}
	if _, err := executeRoot(root, "reserve", "setup", "--income", "5000.00", "--save-rate", "10"); err != nil {
		t.Fatalf("reserve setup: %v", err)
	}

	loadedSubscriptions, err := subscriptionStore.Load()
	if err != nil {
		t.Fatalf("load subscriptions: %v", err)
	}
	if len(loadedSubscriptions.Subscriptions) != 1 || loadedSubscriptions.Subscriptions[0].Name != "Music" {
		t.Fatalf("subscriptions changed: %+v", loadedSubscriptions)
	}
}
