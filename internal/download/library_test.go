package download

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveLibraryAssetPathRejectsTraversal(t *testing.T) {
	root := t.TempDir()

	if _, err := ResolveLibraryAssetPath(root, "/library-files/../secret.jpg"); err == nil {
		t.Fatal("expected traversal path to be rejected")
	}
}

func TestListLibraryMangaAndReaderManifest(t *testing.T) {
	root := t.TempDir()
	mangaDir := filepath.Join(root, "Sample Manga")
	chapterDir := filepath.Join(mangaDir, "001 - Chapter 1")

	if err := os.MkdirAll(chapterDir, 0o755); err != nil {
		t.Fatalf("mkdir chapter dir: %v", err)
	}

	if err := os.WriteFile(filepath.Join(chapterDir, "001.jpg"), []byte("one"), 0o644); err != nil {
		t.Fatalf("write page 1: %v", err)
	}
	if err := os.WriteFile(filepath.Join(chapterDir, "002.jpg"), []byte("two"), 0o644); err != nil {
		t.Fatalf("write page 2: %v", err)
	}

	if err := WriteSidecar(SidecarPath(chapterDir), ChapterSidecar{
		MangaTitle:        "Sample Manga",
		ChapterID:         "chapter-1",
		ChapterNumber:     1,
		ChapterTitle:      "Chapter 1",
		ExpectedPageCount: 2,
		DownloadedPages:   2,
		Files: []ChapterSidecarFile{
			{PageIndex: 0, FileName: "001.jpg"},
			{PageIndex: 1, FileName: "002.jpg"},
		},
		CompletedAt: "2026-04-19T00:00:00Z",
	}); err != nil {
		t.Fatalf("write sidecar: %v", err)
	}

	library, err := ListLibraryManga(root)
	if err != nil {
		t.Fatalf("ListLibraryManga returned error: %v", err)
	}

	if len(library) != 1 {
		t.Fatalf("unexpected library size: %d", len(library))
	}

	if library[0].PageCount != 2 {
		t.Fatalf("unexpected page count: %d", library[0].PageCount)
	}

	if library[0].CoverImageURL == "" {
		t.Fatal("expected cover image url")
	}

	manifest, err := GetReaderManifest(root, library[0].ID)
	if err != nil {
		t.Fatalf("GetReaderManifest returned error: %v", err)
	}

	if manifest.TotalPages != 2 {
		t.Fatalf("unexpected total pages: %d", manifest.TotalPages)
	}

	if len(manifest.Chapters) != 1 || len(manifest.Chapters[0].Pages) != 2 {
		t.Fatalf("unexpected manifest shape: %#v", manifest.Chapters)
	}
}
