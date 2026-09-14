package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Wather17/higher/internal/platform"
	"github.com/Wather17/higher/internal/subscription"
)

const subscriptionsFileName = "subscriptions.json"

type Store struct {
	path string
}

func NewStore(path string) (*Store, error) {
	if path == "" {
		return nil, errors.New("store path cannot be empty")
	}
	return &Store{path: path}, nil
}

func NewDefaultStore() (*Store, error) {
	directory, err := platform.DefaultDataDir()
	if err != nil {
		return nil, err
	}
	return NewStore(filepath.Join(directory, subscriptionsFileName))
}

func (store *Store) Path() string {
	return store.path
}

func (store *Store) Load() (subscription.Document, error) {
	data, err := os.ReadFile(store.path)
	if errors.Is(err, os.ErrNotExist) {
		return subscription.NewDocument(), nil
	}
	if err != nil {
		return subscription.Document{}, fmt.Errorf("read subscriptions file: %w", err)
	}

	var document subscription.Document
	if err := json.Unmarshal(data, &document); err != nil {
		return subscription.Document{}, fmt.Errorf("decode subscriptions file: %w", err)
	}
	if err := document.Validate(); err != nil {
		return subscription.Document{}, err
	}

	return document, nil
}

func (store *Store) Save(document subscription.Document) error {
	if err := document.Validate(); err != nil {
		return err
	}

	directory := filepath.Dir(store.path)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return fmt.Errorf("create data directory: %w", err)
	}

	data, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return fmt.Errorf("encode subscriptions file: %w", err)
	}
	data = append(data, '\n')

	temporary, err := os.CreateTemp(directory, ".subscriptions-*.json")
	if err != nil {
		return fmt.Errorf("create temporary subscriptions file: %w", err)
	}
	temporaryPath := temporary.Name()
	cleanup := func() {
		_ = temporary.Close()
		_ = os.Remove(temporaryPath)
	}

	if err := temporary.Chmod(0o600); err != nil {
		cleanup()
		return fmt.Errorf("set subscriptions file permissions: %w", err)
	}
	if _, err := temporary.Write(data); err != nil {
		cleanup()
		return fmt.Errorf("write subscriptions file: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		cleanup()
		return fmt.Errorf("sync subscriptions file: %w", err)
	}
	if err := temporary.Close(); err != nil {
		_ = os.Remove(temporaryPath)
		return fmt.Errorf("close subscriptions file: %w", err)
	}
	if err := os.Rename(temporaryPath, store.path); err != nil {
		_ = os.Remove(temporaryPath)
		return fmt.Errorf("replace subscriptions file: %w", err)
	}

	return nil
}
