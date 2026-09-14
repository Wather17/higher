package subscription

import (
	"errors"
	"testing"
	"time"
)

func TestParseAmountUsesCents(t *testing.T) {
	tests := map[string]int64{
		"55.90": 5590,
		"10":    1000,
		"0.50":  50,
	}

	for input, expected := range tests {
		amount, err := ParseAmount(input)
		if err != nil {
			t.Fatalf("ParseAmount(%q): %v", input, err)
		}
		if amount != expected {
			t.Fatalf("ParseAmount(%q) = %d, expected %d", input, amount, expected)
		}
	}
}

func TestParseAmountRejectsInvalidValues(t *testing.T) {
	for _, input := range []string{"", "0", "-1.00", "1.001", "1,00", "one"} {
		if _, err := ParseAmount(input); !errors.Is(err, ErrInvalidAmount) {
			t.Errorf("ParseAmount(%q) error = %v, expected ErrInvalidAmount", input, err)
		}
	}
}

func TestCalculateTotalsSeparatesCurrenciesAndNormalizesPeriods(t *testing.T) {
	totals, err := CalculateTotals([]Subscription{
		{AmountCents: 5590, Currency: CurrencyBRL, Period: PeriodMonthly, Status: StatusActive},
		{AmountCents: 12000, Currency: CurrencyBRL, Period: PeriodYearly, Status: StatusActive},
		{AmountCents: 1000, Currency: CurrencyUSD, Period: PeriodMonthly, Status: StatusActive},
		{AmountCents: 5000, Currency: CurrencyBRL, Period: PeriodMonthly, Status: StatusCanceled},
	})
	if err != nil {
		t.Fatalf("CalculateTotals: %v", err)
	}

	if got := totals[CurrencyBRL]; got.MonthlyCents != 6590 || got.AnnualCents != 79080 {
		t.Fatalf("unexpected BRL totals: %+v", got)
	}
	if got := totals[CurrencyUSD]; got.MonthlyCents != 1000 || got.AnnualCents != 12000 {
		t.Fatalf("unexpected USD totals: %+v", got)
	}
}

func TestEditRecordsPriceHistory(t *testing.T) {
	document := NewDocument()
	item, err := document.Add(AddInput{
		Name:        "Streaming",
		AmountCents: 4990,
		Currency:    CurrencyBRL,
		Period:      PeriodMonthly,
	})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	newAmount := int64(5590)
	updated, err := document.Edit(item.ID, EditInput{AmountCents: &newAmount}, time.Date(2026, 9, 14, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("Edit: %v", err)
	}
	if len(updated.PriceHistory) != 1 {
		t.Fatalf("expected one price history entry, got %d", len(updated.PriceHistory))
	}
	change := updated.PriceHistory[0]
	if change.PreviousAmountCents != 4990 || change.NewAmountCents != 5590 || change.ChangedAt != "2026-09-14" {
		t.Fatalf("unexpected price history: %+v", change)
	}
}

func TestCancelAndReactivatePreserveStatusDates(t *testing.T) {
	document := NewDocument()
	item, err := document.Add(AddInput{Name: "Cloud", AmountCents: 1000, Period: PeriodMonthly})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	canceled, err := document.Cancel(item.ID, time.Date(2026, 9, 14, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	if canceled.Status != StatusCanceled || canceled.CanceledAt != "2026-09-14" {
		t.Fatalf("unexpected canceled subscription: %+v", canceled)
	}

	reactivated, err := document.Reactivate(item.ID, time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("Reactivate: %v", err)
	}
	if reactivated.Status != StatusActive || reactivated.CanceledAt != "2026-09-14" || reactivated.ReactivatedAt != "2026-10-01" {
		t.Fatalf("unexpected reactivated subscription: %+v", reactivated)
	}
}

func TestDocumentValidationRejectsDuplicateIDs(t *testing.T) {
	document := NewDocument()
	document.Subscriptions = []Subscription{
		{ID: 1, Name: "One", AmountCents: 100, Currency: CurrencyBRL, Period: PeriodMonthly, Status: StatusActive},
		{ID: 1, Name: "Two", AmountCents: 100, Currency: CurrencyBRL, Period: PeriodMonthly, Status: StatusActive},
	}

	if err := document.Validate(); !errors.Is(err, ErrInvalidDocument) {
		t.Fatalf("expected ErrInvalidDocument, got %v", err)
	}
}
