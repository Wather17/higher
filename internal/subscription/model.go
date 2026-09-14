package subscription

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

const CurrentDocumentVersion = 1

const dateLayout = "2006-01-02"

// MaxAmountCents keeps monthly-to-annual calculations within int64.
const MaxAmountCents = math.MaxInt64 / 12

type Currency string

const (
	CurrencyBRL Currency = "BRL"
	CurrencyUSD Currency = "USD"
)

var SupportedCurrencies = []Currency{CurrencyBRL, CurrencyUSD}

type Period string

const (
	PeriodMonthly Period = "monthly"
	PeriodYearly  Period = "yearly"
)

type Status string

const (
	StatusActive   Status = "active"
	StatusCanceled Status = "canceled"
)

var (
	ErrNotFound          = errors.New("subscription not found")
	ErrAlreadyActive     = errors.New("subscription is already active")
	ErrAlreadyCanceled   = errors.New("subscription is already canceled")
	ErrNoChanges         = errors.New("no changes were provided")
	ErrInvalidDocument   = errors.New("invalid subscriptions document")
	ErrInvalidName       = errors.New("subscription name cannot be empty")
	ErrInvalidAmount     = errors.New("amount must be a positive decimal with at most two decimal places")
	ErrInvalidCurrency   = errors.New("currency must be BRL or USD")
	ErrInvalidPeriod     = errors.New("period must be monthly or yearly")
	ErrInvalidDate       = errors.New("date must use the YYYY-MM-DD format")
	ErrConflictingFields = errors.New("conflicting edit fields")
)

type Document struct {
	Version       int            `json:"version"`
	NextID        int            `json:"next_id"`
	Subscriptions []Subscription `json:"subscriptions"`
}

type Subscription struct {
	ID            int           `json:"id"`
	Name          string        `json:"name"`
	AmountCents   int64         `json:"amount_cents"`
	Currency      Currency      `json:"currency"`
	Period        Period        `json:"period"`
	NextCharge    string        `json:"next_charge,omitempty"`
	Note          string        `json:"note,omitempty"`
	Status        Status        `json:"status"`
	CanceledAt    string        `json:"canceled_at,omitempty"`
	ReactivatedAt string        `json:"reactivated_at,omitempty"`
	PriceHistory  []PriceChange `json:"price_history,omitempty"`
}

type PriceChange struct {
	PreviousAmountCents int64    `json:"previous_amount_cents"`
	PreviousCurrency    Currency `json:"previous_currency"`
	NewAmountCents      int64    `json:"new_amount_cents"`
	NewCurrency         Currency `json:"new_currency"`
	ChangedAt           string   `json:"changed_at"`
}

type AddInput struct {
	Name        string
	AmountCents int64
	Currency    Currency
	Period      Period
	NextCharge  string
	Note        string
}

type EditInput struct {
	Name            *string
	AmountCents     *int64
	Currency        *Currency
	Period          *Period
	NextCharge      *string
	ClearNextCharge bool
	Note            *string
	ClearNote       bool
}

type Totals struct {
	MonthlyCents int64
	AnnualCents  int64
}

func NewDocument() Document {
	return Document{
		Version:       CurrentDocumentVersion,
		NextID:        1,
		Subscriptions: []Subscription{},
	}
}

func (document Document) Validate() error {
	if document.Version != CurrentDocumentVersion {
		return fmt.Errorf("%w: unsupported version %d", ErrInvalidDocument, document.Version)
	}
	if document.NextID < 1 {
		return fmt.Errorf("%w: next_id must be positive", ErrInvalidDocument)
	}

	seenIDs := make(map[int]struct{}, len(document.Subscriptions))
	for _, item := range document.Subscriptions {
		if _, exists := seenIDs[item.ID]; exists {
			return fmt.Errorf("%w: duplicate subscription id %d", ErrInvalidDocument, item.ID)
		}
		seenIDs[item.ID] = struct{}{}
		if err := item.Validate(); err != nil {
			return fmt.Errorf("%w: subscription %d: %v", ErrInvalidDocument, item.ID, err)
		}
	}

	return nil
}

func (item Subscription) Validate() error {
	if item.ID < 1 {
		return errors.New("id must be positive")
	}
	if err := ValidateName(item.Name); err != nil {
		return err
	}
	if err := ValidateAmount(item.AmountCents); err != nil {
		return err
	}
	if err := ValidateCurrency(item.Currency); err != nil {
		return err
	}
	if err := ValidatePeriod(item.Period); err != nil {
		return err
	}
	if item.NextCharge != "" {
		if err := ValidateDate(item.NextCharge); err != nil {
			return fmt.Errorf("next charge: %w", err)
		}
	}
	if item.Status != StatusActive && item.Status != StatusCanceled {
		return fmt.Errorf("invalid status %q", item.Status)
	}
	if item.CanceledAt != "" {
		if err := ValidateDate(item.CanceledAt); err != nil {
			return fmt.Errorf("canceled at: %w", err)
		}
	}
	if item.ReactivatedAt != "" {
		if err := ValidateDate(item.ReactivatedAt); err != nil {
			return fmt.Errorf("reactivated at: %w", err)
		}
	}
	for _, change := range item.PriceHistory {
		if err := change.Validate(); err != nil {
			return fmt.Errorf("price history: %w", err)
		}
	}

	return nil
}

func (change PriceChange) Validate() error {
	if err := ValidateAmount(change.PreviousAmountCents); err != nil {
		return fmt.Errorf("previous amount: %w", err)
	}
	if err := ValidateCurrency(change.PreviousCurrency); err != nil {
		return fmt.Errorf("previous currency: %w", err)
	}
	if err := ValidateAmount(change.NewAmountCents); err != nil {
		return fmt.Errorf("new amount: %w", err)
	}
	if err := ValidateCurrency(change.NewCurrency); err != nil {
		return fmt.Errorf("new currency: %w", err)
	}
	if err := ValidateDate(change.ChangedAt); err != nil {
		return fmt.Errorf("changed at: %w", err)
	}

	return nil
}

func ValidateName(name string) error {
	if strings.TrimSpace(name) == "" {
		return ErrInvalidName
	}
	return nil
}

func ValidateAmount(amountCents int64) error {
	if amountCents <= 0 || amountCents > MaxAmountCents {
		return ErrInvalidAmount
	}
	return nil
}

func ValidateCurrency(currency Currency) error {
	switch currency {
	case CurrencyBRL, CurrencyUSD:
		return nil
	default:
		return fmt.Errorf("%w: %q", ErrInvalidCurrency, currency)
	}
}

func ParseCurrency(value string) (Currency, error) {
	currency := Currency(strings.ToUpper(strings.TrimSpace(value)))
	if err := ValidateCurrency(currency); err != nil {
		return "", err
	}
	return currency, nil
}

func ValidatePeriod(period Period) error {
	if period != PeriodMonthly && period != PeriodYearly {
		return fmt.Errorf("%w: %q", ErrInvalidPeriod, period)
	}
	return nil
}

func ParsePeriod(value string) (Period, error) {
	period := Period(strings.ToLower(strings.TrimSpace(value)))
	if err := ValidatePeriod(period); err != nil {
		return "", err
	}
	return period, nil
}

func ParseAmount(value string) (int64, error) {
	if value == "" || strings.Contains(value, ",") || strings.HasPrefix(value, "-") {
		return 0, ErrInvalidAmount
	}

	parts := strings.Split(value, ".")
	if len(parts) > 2 || parts[0] == "" {
		return 0, ErrInvalidAmount
	}
	if len(parts) == 1 {
		parts = append(parts, "")
	}
	if len(parts[1]) > 2 {
		return 0, ErrInvalidAmount
	}
	for _, part := range parts {
		if part == "" {
			continue
		}
		for _, character := range part {
			if character < '0' || character > '9' {
				return 0, ErrInvalidAmount
			}
		}
	}

	fraction := parts[1]
	for len(fraction) < 2 {
		fraction += "0"
	}
	whole, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || whole > MaxAmountCents/100 {
		return 0, ErrInvalidAmount
	}
	minor, err := strconv.ParseInt(fraction, 10, 64)
	if err != nil {
		return 0, ErrInvalidAmount
	}

	amountCents := whole*100 + minor
	if err := ValidateAmount(amountCents); err != nil {
		return 0, err
	}
	return amountCents, nil
}

func FormatAmount(amountCents int64) string {
	return fmt.Sprintf("%d.%02d", amountCents/100, amountCents%100)
}

func ValidateDate(value string) error {
	if value == "" {
		return nil
	}
	parsed, err := time.Parse(dateLayout, value)
	if err != nil || parsed.Format(dateLayout) != value {
		return fmt.Errorf("%w: %q", ErrInvalidDate, value)
	}
	return nil
}

func DateString(now time.Time) string {
	return now.Format(dateLayout)
}

func (document *Document) Add(input AddInput) (Subscription, error) {
	if err := ValidateName(input.Name); err != nil {
		return Subscription{}, err
	}
	if err := ValidateAmount(input.AmountCents); err != nil {
		return Subscription{}, err
	}
	if input.Currency == "" {
		input.Currency = CurrencyBRL
	}
	if err := ValidateCurrency(input.Currency); err != nil {
		return Subscription{}, err
	}
	if err := ValidatePeriod(input.Period); err != nil {
		return Subscription{}, err
	}
	if err := ValidateDate(input.NextCharge); err != nil {
		return Subscription{}, fmt.Errorf("next charge: %w", err)
	}

	if document.NextID < 1 || hasID(document.Subscriptions, document.NextID) {
		document.NextID = nextAvailableID(document.Subscriptions)
	}
	item := Subscription{
		ID:          document.NextID,
		Name:        strings.TrimSpace(input.Name),
		AmountCents: input.AmountCents,
		Currency:    input.Currency,
		Period:      input.Period,
		NextCharge:  input.NextCharge,
		Note:        strings.TrimSpace(input.Note),
		Status:      StatusActive,
	}
	document.Subscriptions = append(document.Subscriptions, item)
	document.NextID++

	return item, nil
}

func (document *Document) Edit(id int, input EditInput, now time.Time) (Subscription, error) {
	if input.NextCharge != nil && input.ClearNextCharge || input.Note != nil && input.ClearNote {
		return Subscription{}, ErrConflictingFields
	}
	if input.Name == nil && input.AmountCents == nil && input.Currency == nil && input.Period == nil && input.NextCharge == nil && !input.ClearNextCharge && input.Note == nil && !input.ClearNote {
		return Subscription{}, ErrNoChanges
	}

	item, err := document.Find(id)
	if err != nil {
		return Subscription{}, err
	}
	updated := *item

	originalAmount := updated.AmountCents
	originalCurrency := updated.Currency
	changed := false

	if input.Name != nil {
		if err := ValidateName(*input.Name); err != nil {
			return Subscription{}, err
		}
		name := strings.TrimSpace(*input.Name)
		if updated.Name != name {
			updated.Name = name
			changed = true
		}
	}
	if input.AmountCents != nil {
		if err := ValidateAmount(*input.AmountCents); err != nil {
			return Subscription{}, err
		}
		if updated.AmountCents != *input.AmountCents {
			updated.AmountCents = *input.AmountCents
			changed = true
		}
	}
	if input.Currency != nil {
		if err := ValidateCurrency(*input.Currency); err != nil {
			return Subscription{}, err
		}
		if updated.Currency != *input.Currency {
			updated.Currency = *input.Currency
			changed = true
		}
	}
	if input.Period != nil {
		if err := ValidatePeriod(*input.Period); err != nil {
			return Subscription{}, err
		}
		if updated.Period != *input.Period {
			updated.Period = *input.Period
			changed = true
		}
	}
	if input.NextCharge != nil {
		if err := ValidateDate(*input.NextCharge); err != nil {
			return Subscription{}, fmt.Errorf("next charge: %w", err)
		}
		if updated.NextCharge != *input.NextCharge {
			updated.NextCharge = *input.NextCharge
			changed = true
		}
	}
	if input.ClearNextCharge && updated.NextCharge != "" {
		updated.NextCharge = ""
		changed = true
	}
	if input.Note != nil {
		note := strings.TrimSpace(*input.Note)
		if updated.Note != note {
			updated.Note = note
			changed = true
		}
	}
	if input.ClearNote && updated.Note != "" {
		updated.Note = ""
		changed = true
	}
	if !changed {
		return Subscription{}, ErrNoChanges
	}

	if updated.AmountCents != originalAmount || updated.Currency != originalCurrency {
		updated.PriceHistory = append(updated.PriceHistory, PriceChange{
			PreviousAmountCents: originalAmount,
			PreviousCurrency:    originalCurrency,
			NewAmountCents:      updated.AmountCents,
			NewCurrency:         updated.Currency,
			ChangedAt:           DateString(now),
		})
	}

	*item = updated
	return updated, nil
}

func (document *Document) Cancel(id int, now time.Time) (Subscription, error) {
	item, err := document.Find(id)
	if err != nil {
		return Subscription{}, err
	}
	if item.Status == StatusCanceled {
		return Subscription{}, ErrAlreadyCanceled
	}
	item.Status = StatusCanceled
	item.CanceledAt = DateString(now)
	return *item, nil
}

func (document *Document) Reactivate(id int, now time.Time) (Subscription, error) {
	item, err := document.Find(id)
	if err != nil {
		return Subscription{}, err
	}
	if item.Status == StatusActive {
		return Subscription{}, ErrAlreadyActive
	}
	item.Status = StatusActive
	item.ReactivatedAt = DateString(now)
	return *item, nil
}

func (document *Document) Find(id int) (*Subscription, error) {
	for index := range document.Subscriptions {
		if document.Subscriptions[index].ID == id {
			return &document.Subscriptions[index], nil
		}
	}
	return nil, fmt.Errorf("%w: %d", ErrNotFound, id)
}

func CalculateTotals(items []Subscription) (map[Currency]Totals, error) {
	totals := make(map[Currency]Totals)
	for _, item := range items {
		if item.Status != StatusActive {
			continue
		}
		current := totals[item.Currency]
		switch item.Period {
		case PeriodMonthly:
			if item.AmountCents > MaxAmountCents || current.AnnualCents > math.MaxInt64-item.AmountCents*12 {
				return nil, ErrInvalidAmount
			}
			current.MonthlyCents += item.AmountCents
			current.AnnualCents += item.AmountCents * 12
		case PeriodYearly:
			monthly, err := roundAnnualToMonthly(item.AmountCents)
			if err != nil || current.MonthlyCents > math.MaxInt64-monthly || current.AnnualCents > math.MaxInt64-item.AmountCents {
				return nil, ErrInvalidAmount
			}
			current.MonthlyCents += monthly
			current.AnnualCents += item.AmountCents
		default:
			return nil, fmt.Errorf("%w: %q", ErrInvalidPeriod, item.Period)
		}
		totals[item.Currency] = current
	}
	return totals, nil
}

func roundAnnualToMonthly(amountCents int64) (int64, error) {
	if err := ValidateAmount(amountCents); err != nil {
		return 0, err
	}
	monthly := amountCents / 12
	if amountCents%12 >= 6 {
		monthly++
	}
	return monthly, nil
}

func nextAvailableID(items []Subscription) int {
	maximum := 0
	for _, item := range items {
		if item.ID > maximum {
			maximum = item.ID
		}
	}
	return maximum + 1
}

func hasID(items []Subscription, id int) bool {
	for _, item := range items {
		if item.ID == id {
			return true
		}
	}
	return false
}
