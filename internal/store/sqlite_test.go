package store

import (
	"testing"

	"github.com/sakagamijun/rawmanga-download-go/internal/contracts"
)

func TestReaderProgressRoundTrip(t *testing.T) {
	sqliteStore, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("open sqlite store: %v", err)
	}
	defer sqliteStore.Close()

	progress := contracts.ReaderProgress{
		MangaID:   "manga-1",
		ChapterID: "chapter-7",
		Page:      42,
		UpdatedAt: "2026-04-21T00:00:00Z",
	}

	if err := sqliteStore.SaveReaderProgress(progress); err != nil {
		t.Fatalf("save reader progress: %v", err)
	}

	saved, found, err := sqliteStore.GetReaderProgress(progress.MangaID)
	if err != nil {
		t.Fatalf("get reader progress: %v", err)
	}
	if !found {
		t.Fatal("expected reader progress to be found")
	}
	if saved != progress {
		t.Fatalf("unexpected reader progress: %#v", saved)
	}
}
