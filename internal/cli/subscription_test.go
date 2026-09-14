package cli

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Wather17/higher/internal/storage"
	"github.com/spf13/cobra"
)

func TestSubscriptionCommandsPersistAndListSeparateCurrencyTotals(t *testing.T) {
	store, err := storage.NewStore(filepath.Join(t.TempDir(), "subscriptions.json"))
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	root := NewRootCommand(Dependencies{
		Store: store,
		Now:   func() time.Time { return time.Date(2026, 9, 14, 0, 0, 0, 0, time.UTC) },
	})

	if output, err := executeRoot(root, "subscription", "add", "--name", "Netflix", "--amount", "55.90", "--period", "monthly", "--next-charge", "2026-10-01"); err != nil {
		t.Fatalf("add BRL: %v; output: %s", err, output)
	}
	if output, err := executeRoot(root, "subscription", "add", "--name", "Cloud", "--amount", "120.00", "--currency", "USD", "--period", "yearly"); err != nil {
		t.Fatalf("add USD: %v; output: %s", err, output)
	}

	output, err := executeRoot(root, "subscription", "list")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	for _, expected := range []string{"Netflix", "BRL 55.90", "Cloud", "USD 120.00", "Totais mensais (ativas):", "BRL 55.90", "USD 10.00", "Totais anuais (ativas):", "BRL 670.80", "USD 120.00"} {
		if !strings.Contains(output, expected) {
			t.Errorf("list output does not contain %q:\n%s", expected, output)
		}
	}
}

func TestSubscriptionCommandsCancelAndReactivate(t *testing.T) {
	store, err := storage.NewStore(filepath.Join(t.TempDir(), "subscriptions.json"))
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	root := NewRootCommand(Dependencies{
		Store: store,
		Now:   func() time.Time { return time.Date(2026, 9, 14, 0, 0, 0, 0, time.UTC) },
	})
	if _, err := executeRoot(root, "subscription", "add", "--name", "Music", "--amount", "10.00", "--period", "monthly"); err != nil {
		t.Fatalf("add: %v", err)
	}

	if _, err := executeRoot(root, "subscription", "cancel", "--id", "1"); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	output, err := executeRoot(root, "subscription", "list")
	if err != nil {
		t.Fatalf("active list: %v", err)
	}
	if !strings.Contains(output, "Nenhuma assinatura encontrada.") || !strings.Contains(output, "Nenhuma assinatura ativa.") {
		t.Fatalf("expected canceled subscription to leave active list without totals, got:\n%s", output)
	}

	output, err = executeRoot(root, "subscription", "list", "--all")
	if err != nil {
		t.Fatalf("all list: %v", err)
	}
	if !strings.Contains(output, "canceled") {
		t.Fatalf("expected canceled subscription in --all output, got:\n%s", output)
	}

	if _, err := executeRoot(root, "subscription", "reactivate", "--id", "1"); err != nil {
		t.Fatalf("reactivate: %v", err)
	}
	output, err = executeRoot(root, "subscription", "list")
	if err != nil {
		t.Fatalf("reactivated list: %v", err)
	}
	if !strings.Contains(output, "Music") || !strings.Contains(output, "BRL 10.00") {
		t.Fatalf("expected reactivated subscription in active list, got:\n%s", output)
	}
}

func TestSubscriptionAddRejectsInvalidInput(t *testing.T) {
	store, err := storage.NewStore(filepath.Join(t.TempDir(), "subscriptions.json"))
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	root := NewRootCommand(Dependencies{Store: store})

	if _, err := executeRoot(root, "subscription", "add", "--name", "Netflix", "--amount", "1.001", "--period", "monthly"); err == nil {
		t.Fatal("expected invalid amount to fail")
	}
	if _, err := executeRoot(root, "subscription", "add", "--name", "Netflix", "--amount", "10.00", "--currency", "EUR", "--period", "monthly"); err == nil {
		t.Fatal("expected invalid currency to fail")
	}
	if _, err := executeRoot(root, "subscription", "add", "--name", "Netflix", "--amount", "10.00", "--period", "weekly"); err == nil {
		t.Fatal("expected invalid period to fail")
	}
}

func executeRoot(root *cobra.Command, args ...string) (string, error) {
	output := new(bytes.Buffer)
	root.SetOut(output)
	root.SetErr(output)
	root.SetArgs(args)
	err := root.Execute()
	return output.String(), err
}
