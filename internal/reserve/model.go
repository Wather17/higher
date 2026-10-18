package reserve

import (
	"errors"
	"fmt"
	"math"
	"math/big"
	"sort"
	"strconv"
	"strings"
	"time"
)

const CurrentDocumentVersion = 1

const (
	dateLayout              = "2006-01-02"
	recommendedTargetMonths = int64(6)
	maxRateBasisPoints      = int64(10000)
	maxAmountCents          = math.MaxInt64 / recommendedTargetMonths
)

type EntryKind string

const (
	EntryDeposit    EntryKind = "deposit"
	EntryWithdrawal EntryKind = "withdrawal"
)

var (
	ErrNotConfigured       = errors.New("reserve is not configured")
	ErrAlreadyConfigured   = errors.New("reserve is already configured")
	ErrNoChanges           = errors.New("no changes were provided")
	ErrInvalidDocument     = errors.New("invalid reserve document")
	ErrInvalidIncome       = errors.New("income must be a positive decimal with at most two decimal places")
	ErrInvalidAmount       = errors.New("amount must be a positive decimal with at most two decimal places")
	ErrInvalidTarget       = errors.New("target must be a positive decimal with at most two decimal places")
	ErrInvalidRate         = errors.New("saving rate must be between 0.01 and 100.00 percent")
	ErrInvalidDate         = errors.New("date must use the YYYY-MM-DD format")
	ErrInvalidEntryKind    = errors.New("invalid reserve entry kind")
	ErrInsufficientBalance = errors.New("withdrawal exceeds the reserve balance")
)

type Configuration struct {
	IncomeCents         int64 `json:"income_cents"`
	SaveRateBasisPoints int64 `json:"save_rate_basis_points"`
	TargetCents         int64 `json:"target_cents"`
}

type Entry struct {
	ID          int       `json:"id"`
	Kind        EntryKind `json:"kind"`
	AmountCents int64     `json:"amount_cents"`
	Date        string    `json:"date"`
	Note        string    `json:"note,omitempty"`
}

type Document struct {
	Version       int            `json:"version"`
	NextID        int            `json:"next_id"`
	Configuration *Configuration `json:"configuration,omitempty"`
	Entries       []Entry        `json:"entries"`
}

type SetupInput struct {
	IncomeCents         int64
	SaveRateBasisPoints int64
	TargetCents         *int64
}

type EditInput struct {
	IncomeCents         *int64
	SaveRateBasisPoints *int64
	TargetCents         *int64
}

type EntryInput struct {
	AmountCents int64
	Date        string
	Note        string
}

type Summary struct {
	IncomeCents         int64
	SaveRateBasisPoints int64
	MonthlyContribution int64
	TargetCents         int64
	BalanceCents        int64
	RemainingCents      int64
	ProgressBasisPoints int64
	MonthsRemaining     int64
	Achieved            bool
}

func NewDocument() Document {
	return Document{
		Version: CurrentDocumentVersion,
		NextID:  1,
		Entries: []Entry{},
	}
}

func (document Document) IsConfigured() bool {
	return document.Configuration != nil
}

func (document Document) Validate() error {
	if document.Version != CurrentDocumentVersion {
		return fmt.Errorf("%w: unsupported version %d", ErrInvalidDocument, document.Version)
	}
	if document.NextID < 1 {
		return fmt.Errorf("%w: next_id must be positive", ErrInvalidDocument)
	}
	if document.Configuration == nil && len(document.Entries) > 0 {
		return fmt.Errorf("%w: entries require a configuration", ErrInvalidDocument)
	}
	if document.Configuration != nil {
		if err := document.Configuration.Validate(); err != nil {
			return fmt.Errorf("%w: configuration: %v", ErrInvalidDocument, err)
		}
	}

	seenIDs := make(map[int]struct{}, len(document.Entries))
	balance := int64(0)
	for _, entry := range document.Entries {
		if _, exists := seenIDs[entry.ID]; exists {
			return fmt.Errorf("%w: duplicate entry id %d", ErrInvalidDocument, entry.ID)
		}
		seenIDs[entry.ID] = struct{}{}
		if err := entry.Validate(); err != nil {
			return fmt.Errorf("%w: entry %d: %v", ErrInvalidDocument, entry.ID, err)
		}
		var err error
		switch entry.Kind {
		case EntryDeposit:
			if balance > math.MaxInt64-entry.AmountCents {
				return fmt.Errorf("%w: balance overflow", ErrInvalidDocument)
			}
			balance += entry.AmountCents
		case EntryWithdrawal:
			if entry.AmountCents > balance {
				return fmt.Errorf("%w: entry %d exceeds balance", ErrInvalidDocument, entry.ID)
			}
			balance -= entry.AmountCents
		default:
			err = ErrInvalidEntryKind
		}
		if err != nil {
			return fmt.Errorf("%w: entry %d: %v", ErrInvalidDocument, entry.ID, err)
		}
	}

	return nil
}

func (configuration Configuration) Validate() error {
	if configuration.IncomeCents <= 0 || configuration.IncomeCents > maxAmountCents {
		return ErrInvalidIncome
	}
	if configuration.SaveRateBasisPoints < 1 || configuration.SaveRateBasisPoints > maxRateBasisPoints {
		return ErrInvalidRate
	}
	if configuration.TargetCents <= 0 || configuration.TargetCents > maxAmountCents {
		return ErrInvalidTarget
	}
	if _, err := configuration.MonthlyContribution(); err != nil {
		return err
	}
	return nil
}

func (entry Entry) Validate() error {
	if entry.ID < 1 {
		return errors.New("id must be positive")
	}
	if entry.AmountCents <= 0 || entry.AmountCents > maxAmountCents {
		return ErrInvalidAmount
	}
	if err := ValidateDate(entry.Date); err != nil {
		return err
	}
	if entry.Note != strings.TrimSpace(entry.Note) {
		return errors.New("note must be trimmed")
	}
	switch entry.Kind {
	case EntryDeposit, EntryWithdrawal:
		return nil
	default:
		return ErrInvalidEntryKind
	}
}

func ParseRate(value string) (int64, error) {
	value = strings.TrimSpace(value)
	if value == "" || strings.Contains(value, ",") || strings.HasPrefix(value, "-") {
		return 0, ErrInvalidRate
	}

	parts := strings.Split(value, ".")
	if len(parts) > 2 || parts[0] == "" {
		return 0, ErrInvalidRate
	}
	if len(parts) == 1 {
		parts = append(parts, "")
	}
	if len(parts[1]) > 2 {
		return 0, ErrInvalidRate
	}
	for _, part := range parts {
		for _, character := range part {
			if character < '0' || character > '9' {
				return 0, ErrInvalidRate
			}
		}
	}

	fraction := parts[1]
	for len(fraction) < 2 {
		fraction += "0"
	}
	whole, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || whole > 100 {
		return 0, ErrInvalidRate
	}
	minor, err := strconv.ParseInt(fraction, 10, 64)
	if err != nil {
		return 0, ErrInvalidRate
	}
	if whole == 100 && minor > 0 {
		return 0, ErrInvalidRate
	}

	rate := whole*100 + minor
	if rate < 1 || rate > maxRateBasisPoints {
		return 0, ErrInvalidRate
	}
	return rate, nil
}

func FormatRate(rateBasisPoints int64) string {
	return fmt.Sprintf("%d.%02d%%", rateBasisPoints/100, rateBasisPoints%100)
}

func ValidateDate(value string) error {
	if value == "" {
		return ErrInvalidDate
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

func (configuration Configuration) MonthlyContribution() (int64, error) {
	if configuration.IncomeCents <= 0 || configuration.IncomeCents > maxAmountCents {
		return 0, ErrInvalidIncome
	}
	if configuration.SaveRateBasisPoints < 1 || configuration.SaveRateBasisPoints > maxRateBasisPoints {
		return 0, ErrInvalidRate
	}
	quotient := configuration.IncomeCents / 10000
	remainder := configuration.IncomeCents % 10000
	if quotient > math.MaxInt64/configuration.SaveRateBasisPoints {
		return 0, ErrInvalidIncome
	}
	monthly := quotient * configuration.SaveRateBasisPoints
	partial := remainder * configuration.SaveRateBasisPoints
	partial = (partial + 5000) / 10000
	if monthly > math.MaxInt64-partial {
		return 0, ErrInvalidIncome
	}
	monthly += partial
	if monthly < 1 {
		return 0, ErrInvalidRate
	}
	return monthly, nil
}

func (document *Document) Setup(input SetupInput) error {
	if document.Configuration != nil {
		return ErrAlreadyConfigured
	}

	target := input.TargetCents
	if target == nil {
		defaultTarget, err := defaultTarget(input.IncomeCents)
		if err != nil {
			return err
		}
		target = &defaultTarget
	}
	configuration := Configuration{
		IncomeCents:         input.IncomeCents,
		SaveRateBasisPoints: input.SaveRateBasisPoints,
		TargetCents:         *target,
	}
	if err := configuration.Validate(); err != nil {
		return err
	}
	document.Configuration = &configuration
	return nil
}

func (document *Document) Edit(input EditInput) error {
	if document.Configuration == nil {
		return ErrNotConfigured
	}
	if input.IncomeCents == nil && input.SaveRateBasisPoints == nil && input.TargetCents == nil {
		return ErrNoChanges
	}

	updated := *document.Configuration
	if input.IncomeCents != nil {
		updated.IncomeCents = *input.IncomeCents
	}
	if input.SaveRateBasisPoints != nil {
		updated.SaveRateBasisPoints = *input.SaveRateBasisPoints
	}
	if input.TargetCents != nil {
		updated.TargetCents = *input.TargetCents
	}
	if updated == *document.Configuration {
		return ErrNoChanges
	}
	if err := updated.Validate(); err != nil {
		return err
	}
	document.Configuration = &updated
	return nil
}

func (document *Document) Deposit(input EntryInput, now time.Time) (Entry, error) {
	return document.addEntry(EntryDeposit, input, now)
}

func (document *Document) Withdraw(input EntryInput, now time.Time) (Entry, error) {
	return document.addEntry(EntryWithdrawal, input, now)
}

func (document *Document) addEntry(kind EntryKind, input EntryInput, now time.Time) (Entry, error) {
	if document.Configuration == nil {
		return Entry{}, ErrNotConfigured
	}
	if input.AmountCents <= 0 || input.AmountCents > maxAmountCents {
		return Entry{}, ErrInvalidAmount
	}
	date := input.Date
	if date == "" {
		date = DateString(now)
	}
	if err := ValidateDate(date); err != nil {
		return Entry{}, err
	}
	note := strings.TrimSpace(input.Note)
	if kind == EntryWithdrawal {
		balance, err := document.Balance()
		if err != nil {
			return Entry{}, err
		}
		if input.AmountCents > balance {
			return Entry{}, ErrInsufficientBalance
		}
	}

	if document.NextID < 1 || hasID(document.Entries, document.NextID) {
		document.NextID = nextAvailableID(document.Entries)
	}
	entry := Entry{
		ID:          document.NextID,
		Kind:        kind,
		AmountCents: input.AmountCents,
		Date:        date,
		Note:        note,
	}
	document.Entries = append(document.Entries, entry)
	document.NextID++
	return entry, nil
}

func (document Document) Balance() (int64, error) {
	balance := int64(0)
	for _, entry := range document.Entries {
		switch entry.Kind {
		case EntryDeposit:
			if balance > math.MaxInt64-entry.AmountCents {
				return 0, ErrInvalidAmount
			}
			balance += entry.AmountCents
		case EntryWithdrawal:
			if entry.AmountCents > balance {
				return 0, ErrInsufficientBalance
			}
			balance -= entry.AmountCents
		default:
			return 0, ErrInvalidEntryKind
		}
	}
	return balance, nil
}

func (document Document) Summary() (Summary, error) {
	if document.Configuration == nil {
		return Summary{}, ErrNotConfigured
	}
	monthly, err := document.Configuration.MonthlyContribution()
	if err != nil {
		return Summary{}, err
	}
	balance, err := document.Balance()
	if err != nil {
		return Summary{}, err
	}
	remaining := document.Configuration.TargetCents - balance
	if remaining < 0 {
		remaining = 0
	}
	achieved := balance >= document.Configuration.TargetCents
	progress := progressBasisPoints(balance, document.Configuration.TargetCents)
	months := int64(0)
	if !achieved {
		months = remaining / monthly
		if remaining%monthly != 0 {
			months++
		}
	}
	return Summary{
		IncomeCents:         document.Configuration.IncomeCents,
		SaveRateBasisPoints: document.Configuration.SaveRateBasisPoints,
		MonthlyContribution: monthly,
		TargetCents:         document.Configuration.TargetCents,
		BalanceCents:        balance,
		RemainingCents:      remaining,
		ProgressBasisPoints: progress,
		MonthsRemaining:     months,
		Achieved:            achieved,
	}, nil
}

func (document Document) SortedEntries() []Entry {
	entries := append([]Entry(nil), document.Entries...)
	sort.SliceStable(entries, func(left, right int) bool {
		if entries[left].Date == entries[right].Date {
			return entries[left].ID > entries[right].ID
		}
		return entries[left].Date > entries[right].Date
	})
	return entries
}

func defaultTarget(incomeCents int64) (int64, error) {
	if incomeCents <= 0 || incomeCents > maxAmountCents {
		return 0, ErrInvalidIncome
	}
	if incomeCents > math.MaxInt64/recommendedTargetMonths {
		return 0, ErrInvalidTarget
	}
	return incomeCents * recommendedTargetMonths, nil
}

func progressBasisPoints(balance, target int64) int64 {
	if target <= 0 || balance <= 0 {
		return 0
	}
	if balance >= target {
		return maxRateBasisPoints
	}
	numerator := new(big.Int).Mul(big.NewInt(balance), big.NewInt(maxRateBasisPoints))
	numerator.Quo(numerator, big.NewInt(target))
	progress := numerator.Int64()
	if progress > maxRateBasisPoints {
		return maxRateBasisPoints
	}
	return progress
}

func nextAvailableID(entries []Entry) int {
	maximum := 0
	for _, entry := range entries {
		if entry.ID > maximum {
			maximum = entry.ID
		}
	}
	return maximum + 1
}

func hasID(entries []Entry, id int) bool {
	for _, entry := range entries {
		if entry.ID == id {
			return true
		}
	}
	return false
}
