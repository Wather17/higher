package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Wather17/higher/internal/platform"
	"github.com/Wather17/higher/internal/reserve"
)

const reserveFileName = "reserve.json"

type ReserveStore struct {
	path string
}

func NewReserveStore(path string) (*ReserveStore, error) {
	if path == "" {
		return nil, errors.New("reserve store path cannot be empty")
	}
	return &ReserveStore{path: path}, nil
}

func NewDefaultReserveStore() (*ReserveStore, error) {
	directory, err := platform.DefaultDataDir()
	if err != nil {
		return nil, err
	}
	return NewReserveStore(filepath.Join(directory, reserveFileName))
}

func (store *ReserveStore) Path() string {
	return store.path
}

func (store *ReserveStore) Load() (reserve.Document, error) {
	data, err := os.ReadFile(store.path)
	if errors.Is(err, os.ErrNotExist) {
		return reserve.NewDocument(), nil
	}
	if err != nil {
		return reserve.Document{}, fmt.Errorf("read reserve file: %w", err)
	}

	var document reserve.Document
	if err := json.Unmarshal(data, &document); err != nil {
		return reserve.Document{}, fmt.Errorf("decode reserve file: %w", err)
	}
	if err := document.Validate(); err != nil {
		return reserve.Document{}, err
	}

	return document, nil
}

func (store *ReserveStore) Save(document reserve.Document) error {
	if err := document.Validate(); err != nil {
		return err
	}

	directory := filepath.Dir(store.path)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return fmt.Errorf("create reserve data directory: %w", err)
	}

	data, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return fmt.Errorf("encode reserve file: %w", err)
	}
	data = append(data, '\n')

	temporary, err := os.CreateTemp(directory, ".reserve-*.json")
	if err != nil {
		return fmt.Errorf("create temporary reserve file: %w", err)
	}
	temporaryPath := temporary.Name()
	cleanup := func() {
		_ = temporary.Close()
		_ = os.Remove(temporaryPath)
	}

	if err := temporary.Chmod(0o600); err != nil {
		cleanup()
		return fmt.Errorf("set reserve file permissions: %w", err)
	}
	if _, err := temporary.Write(data); err != nil {
		cleanup()
		return fmt.Errorf("write reserve file: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		cleanup()
		return fmt.Errorf("sync reserve file: %w", err)
	}
	if err := temporary.Close(); err != nil {
		_ = os.Remove(temporaryPath)
		return fmt.Errorf("close reserve file: %w", err)
	}
	if err := os.Rename(temporaryPath, store.path); err != nil {
		_ = os.Remove(temporaryPath)
		return fmt.Errorf("replace reserve file: %w", err)
	}

	return nil
}
