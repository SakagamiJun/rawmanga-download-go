package download

import (
	"path/filepath"
	"testing"
)

func TestSidecarRoundTrip(t *testing.T) {
	chapterDir := t.TempDir()
	sidecar := ChapterSidecar{
		SourceURL:         "https://klz9.com/foo.html",
		MangaSlug:         "foo",
		MangaTitle:        "Foo",
		ChapterID:         "1",
		ChapterNumber:     1,
		ChapterTitle:      "Chapter 1",
		BundleHash:        "bundle.js",
		ExpectedPageCount: 12,
		DownloadedPages:   10,
	}

	path := SidecarPath(chapterDir)
	if err := WriteSidecar(path, sidecar); err != nil {
		t.Fatalf("write sidecar: %v", err)
	}

	decoded, found, err := ReadSidecar(path)
	if err != nil {
		t.Fatalf("read sidecar: %v", err)
	}
	if !found {
		t.Fatal("expected sidecar to exist")
	}
	if decoded.ExpectedPageCount != 12 {
		t.Fatalf("unexpected expected pages: %d", decoded.ExpectedPageCount)
	}
}

func TestDirectoryNaming(t *testing.T) {
	root := "/tmp/KLZ9"
	mangaDir := MangaDirectory(root, "A/B:C*D")
	if filepath.Base(mangaDir) != "A B C D" {
		t.Fatalf("unexpected manga directory name: %s", filepath.Base(mangaDir))
	}

	chapterDir := ChapterDirectory(mangaDir, 12.5, "Chapter / 12.5")
	if filepath.Base(chapterDir) != "12_5 - Chapter 12.5" {
		t.Fatalf("unexpected chapter directory name: %s", filepath.Base(chapterDir))
	}
}
