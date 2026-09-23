package reserve

import (
	"errors"
	"testing"
	"time"
)

func TestParseRateUsesBasisPoints(t *testing.T) {
	tests := map[string]int64{
		"10":     1000,
		"10.5":   1050,
		"0.01":   1,
		"100.00": 10000,
	}

	for input, expected := range tests {
		got, err := ParseRate(input)
		if err != nil {
			t.Fatalf("ParseRate(%q): %v", input, err)
		}
		if got != expected {
			t.Fatalf("ParseRate(%q) = %d, expected %d", input, got, expected)
		}
	}
}

func TestParseRateRejectsInvalidValues(t *testing.T) {
	for _, input := range []string{"", "0", "0.00", "100.01", "1.001", "1,00", "-1", "ten"} {
		if _, err := ParseRate(input); !errors.Is(err, ErrInvalidRate) {
			t.Errorf("ParseRate(%q) error = %v, expected ErrInvalidRate", input, err)
		}
	}
}

func TestSetupUsesSixIncomeTargetAndCalculatesMonthlyContribution(t *testing.T) {
	document := NewDocument()
	if err := document.Setup(SetupInput{IncomeCents: 500000, SaveRateBasisPoints: 1000}); err != nil {
		t.Fatalf("Setup: %v", err)
	}

	if document.Configuration.TargetCents != 3000000 {
		t.Fatalf("expected default target 3000000, got %d", document.Configuration.TargetCents)
	}
	monthly, err := document.Configuration.MonthlyContribution()
	if err != nil {
		t.Fatalf("MonthlyContribution: %v", err)
	}
	if monthly != 50000 {
		t.Fatalf("expected monthly contribution 50000, got %d", monthly)
	}
}

func TestMonthlyContributionRoundsToNearestCent(t *testing.T) {
	configuration := Configuration{IncomeCents: 33333, SaveRateBasisPoints: 1000, TargetCents: 1}
	monthly, err := configuration.MonthlyContribution()
	if err != nil {
		t.Fatalf("MonthlyContribution: %v", err)
	}
	if monthly != 3333 {
		t.Fatalf("expected 3333 cents, got %d", monthly)
	}
}

func TestDepositWithdrawAndSummary(t *testing.T) {
	document := NewDocument()
	if err := document.Setup(SetupInput{IncomeCents: 500000, SaveRateBasisPoints: 1000, TargetCents: ptrInt64(100000)}); err != nil {
		t.Fatalf("Setup: %v", err)
	}

	deposit, err := document.Deposit(EntryInput{AmountCents: 60000, Note: "  aporte mensal  "}, time.Date(2026, 9, 23, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("Deposit: %v", err)
	}
	if deposit.ID != 1 || deposit.Date != "2026-09-23" || deposit.Note != "aporte mensal" {
		t.Fatalf("unexpected deposit: %+v", deposit)
	}

	withdrawal, err := document.Withdraw(EntryInput{AmountCents: 10000, Date: "2026-09-24"}, time.Time{})
	if err != nil {
		t.Fatalf("Withdraw: %v", err)
	}
	if withdrawal.ID != 2 || withdrawal.Kind != EntryWithdrawal {
		t.Fatalf("unexpected withdrawal: %+v", withdrawal)
	}

	summary, err := document.Summary()
	if err != nil {
		t.Fatalf("Summary: %v", err)
	}
	if summary.BalanceCents != 50000 || summary.RemainingCents != 50000 || summary.ProgressBasisPoints != 5000 || summary.MonthsRemaining != 1 {
		t.Fatalf("unexpected summary: %+v", summary)
	}
}

func TestWithdrawRejectsInsufficientBalanceWithoutAddingEntry(t *testing.T) {
	document := NewDocument()
	if err := document.Setup(SetupInput{IncomeCents: 500000, SaveRateBasisPoints: 1000}); err != nil {
		t.Fatalf("Setup: %v", err)
	}

	_, err := document.Withdraw(EntryInput{AmountCents: 1}, time.Date(2026, 9, 23, 0, 0, 0, 0, time.UTC))
	if !errors.Is(err, ErrInsufficientBalance) {
		t.Fatalf("Withdraw error = %v, expected ErrInsufficientBalance", err)
	}
	if len(document.Entries) != 0 || document.NextID != 1 {
		t.Fatalf("withdrawal changed document: %+v", document)
	}
}

func TestSummaryReportsAchievedTargetAndCapsProgress(t *testing.T) {
	document := NewDocument()
	if err := document.Setup(SetupInput{IncomeCents: 500000, SaveRateBasisPoints: 1000, TargetCents: ptrInt64(10000)}); err != nil {
		t.Fatalf("Setup: %v", err)
	}
	if _, err := document.Deposit(EntryInput{AmountCents: 15000}, time.Date(2026, 9, 23, 0, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("Deposit: %v", err)
	}

	summary, err := document.Summary()
	if err != nil {
		t.Fatalf("Summary: %v", err)
	}
	if !summary.Achieved || summary.RemainingCents != 0 || summary.ProgressBasisPoints != 10000 || summary.MonthsRemaining != 0 {
		t.Fatalf("unexpected achieved summary: %+v", summary)
	}
}

func TestConfigurationRejectsContributionBelowOneCent(t *testing.T) {
	configuration := Configuration{IncomeCents: 1, SaveRateBasisPoints: 1, TargetCents: 1}
	if err := configuration.Validate(); !errors.Is(err, ErrInvalidRate) {
		t.Fatalf("configuration error = %v, expected ErrInvalidRate", err)
	}
}

func TestEntryUsesNowWhenDateIsOmitted(t *testing.T) {
	document := NewDocument()
	if err := document.Setup(SetupInput{IncomeCents: 500000, SaveRateBasisPoints: 1000}); err != nil {
		t.Fatalf("Setup: %v", err)
	}

	entry, err := document.Deposit(EntryInput{AmountCents: 1000}, time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("Deposit: %v", err)
	}
	if entry.Date != "2026-10-01" {
		t.Fatalf("expected default date 2026-10-01, got %q", entry.Date)
	}
}

func TestEditPreservesEntriesAndRejectsNoChanges(t *testing.T) {
	document := NewDocument()
	if err := document.Setup(SetupInput{IncomeCents: 500000, SaveRateBasisPoints: 1000}); err != nil {
		t.Fatalf("Setup: %v", err)
	}
	if _, err := document.Deposit(EntryInput{AmountCents: 10000}, time.Date(2026, 9, 23, 0, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("Deposit: %v", err)
	}

	if err := document.Edit(EditInput{}); !errors.Is(err, ErrNoChanges) {
		t.Fatalf("Edit error = %v, expected ErrNoChanges", err)
	}
	newRate := int64(1500)
	if err := document.Edit(EditInput{SaveRateBasisPoints: &newRate}); err != nil {
		t.Fatalf("Edit: %v", err)
	}
	if len(document.Entries) != 1 || document.Entries[0].AmountCents != 10000 || document.Configuration.SaveRateBasisPoints != 1500 {
		t.Fatalf("edit did not preserve document: %+v", document)
	}
}

func TestSortedEntriesReturnsNewestFirstWithoutChangingStoredOrder(t *testing.T) {
	document := NewDocument()
	if err := document.Setup(SetupInput{IncomeCents: 500000, SaveRateBasisPoints: 1000}); err != nil {
		t.Fatalf("Setup: %v", err)
	}
	if _, err := document.Deposit(EntryInput{AmountCents: 10000, Date: "2026-09-20"}, time.Time{}); err != nil {
		t.Fatalf("Deposit: %v", err)
	}
	if _, err := document.Deposit(EntryInput{AmountCents: 10000, Date: "2026-09-23"}, time.Time{}); err != nil {
		t.Fatalf("Deposit: %v", err)
	}

	sorted := document.SortedEntries()
	if sorted[0].Date != "2026-09-23" || document.Entries[0].Date != "2026-09-20" {
		t.Fatalf("unexpected ordering: sorted=%+v stored=%+v", sorted, document.Entries)
	}
}

func TestDocumentValidationRejectsDuplicateIDs(t *testing.T) {
	document := NewDocument()
	document.Configuration = &Configuration{IncomeCents: 500000, SaveRateBasisPoints: 1000, TargetCents: 3000000}
	document.Entries = []Entry{
		{ID: 1, Kind: EntryDeposit, AmountCents: 100, Date: "2026-09-23"},
		{ID: 1, Kind: EntryDeposit, AmountCents: 100, Date: "2026-09-24"},
	}

	if err := document.Validate(); !errors.Is(err, ErrInvalidDocument) {
		t.Fatalf("expected ErrInvalidDocument, got %v", err)
	}
}

func ptrInt64(value int64) *int64 {
	return &value
}
