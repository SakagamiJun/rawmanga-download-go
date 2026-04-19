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
