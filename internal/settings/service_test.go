package settings

import (
	"path/filepath"
	"testing"

	"github.com/sakagamijun/rawmanga-download-go/internal/store"
)

func TestNewServicePersistsDefaults(t *testing.T) {
	sqliteStore, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open sqlite store: %v", err)
	}
	defer sqliteStore.Close()

	service, err := NewService(sqliteStore)
	if err != nil {
		t.Fatalf("new settings service: %v", err)
	}

	current := service.Get()
	if current.MaxConcurrentDownloads != 6 {
		t.Fatalf("unexpected default max concurrency: %d", current.MaxConcurrentDownloads)
	}
	if current.LocaleMode != "system" {
		t.Fatalf("unexpected default locale mode: %s", current.LocaleMode)
	}
	if current.ReaderScrollCachePages != 6 {
		t.Fatalf("unexpected default reader cache size: %d", current.ReaderScrollCachePages)
	}
	if !current.AutoRestoreReaderProgress {
		t.Fatal("expected auto restore reader progress to be enabled by default")
	}
	if filepath.Base(current.OutputRoot) != "KLZ9" {
		t.Fatalf("unexpected output root: %s", current.OutputRoot)
	}
}

func TestNormalizeRejectsUnsupportedLocale(t *testing.T) {
	sqliteStore, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open sqlite store: %v", err)
	}
	defer sqliteStore.Close()

	service, err := NewService(sqliteStore)
	if err != nil {
		t.Fatalf("new settings service: %v", err)
	}

	_, err = service.Normalize(DefaultSettings())
	if err != nil {
		t.Fatalf("normalize defaults should succeed: %v", err)
	}

	input := DefaultSettings()
	input.LocaleMode = "manual"
	input.Locale = "fr"
	if _, err := service.Normalize(input); err == nil {
		t.Fatal("expected normalize to reject unsupported locale")
	}
}

func TestNormalizeAppliesReaderDefaults(t *testing.T) {
	sqliteStore, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open sqlite store: %v", err)
	}
	defer sqliteStore.Close()

	service, err := NewService(sqliteStore)
	if err != nil {
		t.Fatalf("new settings service: %v", err)
	}

	input := DefaultSettings()
	input.ReaderScrollCachePages = 12
	input.AutoRestoreReaderProgress = false

	normalized, err := service.Normalize(input)
	if err != nil {
		t.Fatalf("normalize settings: %v", err)
	}

	if normalized.ReaderScrollCachePages != 12 {
		t.Fatalf("expected reader cache pages to keep explicit value, got %d", normalized.ReaderScrollCachePages)
	}
	if normalized.AutoRestoreReaderProgress {
		t.Fatal("expected auto restore reader progress to follow explicit false value")
	}
}
