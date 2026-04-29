package download

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestResolveLibraryAssetPathRejectsTraversal(t *testing.T) {
	root := t.TempDir()

	if _, err := ResolveLibraryAssetPath(root, "/library-files/../secret.jpg"); err == nil {
		t.Fatal("expected traversal path to be rejected")
	}
}

func TestIsLibraryAssetRequest(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{path: "/library-files/a.jpg", want: true},
		{path: "/library-archive/a/b", want: true},
		{path: "/assets/index.js", want: false},
	}

	for _, test := range tests {
		if got := IsLibraryAssetRequest(test.path); got != test.want {
			t.Fatalf("unexpected asset routing match for %q: got %v want %v", test.path, got, test.want)
		}
	}
}

func TestListLibraryMangaAndReaderManifestSupportsDirectoryAndArchiveChapters(t *testing.T) {
	root := t.TempDir()
	mangaDir := filepath.Join(root, "Sample Manga")
	chapterDir := filepath.Join(mangaDir, "001 - Chapter 1")
	archivePath := filepath.Join(mangaDir, "002 - Chapter 2.cbz")

	if err := os.MkdirAll(chapterDir, 0o755); err != nil {
		t.Fatalf("mkdir chapter dir: %v", err)
	}

	writeFile(t, filepath.Join(chapterDir, "001.jpg"), "one")
	writeFile(t, filepath.Join(chapterDir, "002.jpg"), "two")
	if err := WriteSidecar(SidecarPath(chapterDir), ChapterSidecar{
		SourceURL:         "https://klz9.com/sample-manga.html",
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

	writeZipArchive(t, archivePath, map[string]string{
		"images/001.jpg": "three",
		"images/002.jpg": "four",
	})
	if err := WriteSidecar(ArchiveSidecarPath(archivePath), ChapterSidecar{
		SourceURL:         "https://klz9.com/sample-manga.html",
		MangaTitle:        "Sample Manga",
		ChapterID:         "chapter-2",
		ChapterNumber:     2,
		ChapterTitle:      "Chapter 2",
		ExpectedPageCount: 2,
		DownloadedPages:   2,
		Files: []ChapterSidecarFile{
			{PageIndex: 0, FileName: "images/001.jpg"},
			{PageIndex: 1, FileName: "images/002.jpg"},
		},
		CompletedAt: "2026-04-20T00:00:00Z",
	}); err != nil {
		t.Fatalf("write archive sidecar: %v", err)
	}

	library, err := ListLibraryManga(root)
	if err != nil {
		t.Fatalf("ListLibraryManga returned error: %v", err)
	}

	if len(library) != 1 {
		t.Fatalf("unexpected library size: %d", len(library))
	}

	item := library[0]
	if item.PageCount != 4 {
		t.Fatalf("unexpected page count: %d", item.PageCount)
	}
	if item.ChapterCount != 2 {
		t.Fatalf("unexpected chapter count: %d", item.ChapterCount)
	}
	if !strings.HasPrefix(item.CoverImageURL, LibraryAssetPrefix) {
		t.Fatalf("expected filesystem cover image url, got %q", item.CoverImageURL)
	}
	if item.SourceURL != "https://klz9.com/sample-manga.html" {
		t.Fatalf("unexpected source url: %q", item.SourceURL)
	}

	manifest, err := GetReaderManifest(root, item.ID)
	if err != nil {
		t.Fatalf("GetReaderManifest returned error: %v", err)
	}

	if manifest.TotalPages != 4 {
		t.Fatalf("unexpected total pages: %d", manifest.TotalPages)
	}
	if len(manifest.Chapters) != 2 {
		t.Fatalf("unexpected chapter count in manifest: %d", len(manifest.Chapters))
	}
	if manifest.CoverImageURL != manifest.Chapters[0].Pages[0].SourceURL {
		t.Fatalf("unexpected cover image url: %q", manifest.CoverImageURL)
	}

	if !sameResolvedPath(manifest.Chapters[0].LocalPath, chapterDir) {
		t.Fatalf("unexpected directory local path: %q", manifest.Chapters[0].LocalPath)
	}
	if !sameResolvedPath(manifest.Chapters[1].LocalPath, archivePath) {
		t.Fatalf("unexpected archive local path: %q", manifest.Chapters[1].LocalPath)
	}
	if !strings.HasPrefix(manifest.Chapters[1].Pages[0].SourceURL, LibraryArchiveAssetPrefix) {
		t.Fatalf("expected archive page url, got %q", manifest.Chapters[1].Pages[0].SourceURL)
	}
}

func TestGetReaderManifestArchiveSidecarOrderWinsOverLexicalOrder(t *testing.T) {
	root := t.TempDir()
	mangaDir := filepath.Join(root, "Sidecar Manga")
	archivePath := filepath.Join(mangaDir, "010 - Bonus.cbz")

	if err := os.MkdirAll(mangaDir, 0o755); err != nil {
		t.Fatalf("mkdir manga dir: %v", err)
	}

	writeZipArchive(t, archivePath, map[string]string{
		"pages/2.jpg":  "two",
		"pages/10.jpg": "ten",
	})
	if err := WriteSidecar(ArchiveSidecarPath(archivePath), ChapterSidecar{
		ChapterID:     "bonus",
		ChapterNumber: 10,
		ChapterTitle:  "Bonus",
		Files: []ChapterSidecarFile{
			{PageIndex: 0, FileName: "pages/10.jpg"},
			{PageIndex: 1, FileName: "pages/2.jpg"},
		},
	}); err != nil {
		t.Fatalf("write archive sidecar: %v", err)
	}

	manifest, err := GetReaderManifest(root, encodeMangaID("Sidecar Manga"))
	if err != nil {
		t.Fatalf("GetReaderManifest returned error: %v", err)
	}

	if len(manifest.Chapters) != 1 {
		t.Fatalf("unexpected chapter count: %d", len(manifest.Chapters))
	}

	pages := manifest.Chapters[0].Pages
	if len(pages) != 2 {
		t.Fatalf("unexpected page count: %d", len(pages))
	}
	if pages[0].FileName != "pages/10.jpg" || pages[1].FileName != "pages/2.jpg" {
		t.Fatalf("unexpected page order: %#v", pages)
	}
}

func TestGetReaderManifestArchiveFallbackUsesNaturalOrder(t *testing.T) {
	root := t.TempDir()
	mangaDir := filepath.Join(root, "Natural Manga")
	archivePath := filepath.Join(mangaDir, "003 - Natural.cbz")

	if err := os.MkdirAll(mangaDir, 0o755); err != nil {
		t.Fatalf("mkdir manga dir: %v", err)
	}

	writeZipArchive(t, archivePath, map[string]string{
		"10.jpg": "ten",
		"2.jpg":  "two",
		"3.jpg":  "three",
	})

	manifest, err := GetReaderManifest(root, encodeMangaID("Natural Manga"))
	if err != nil {
		t.Fatalf("GetReaderManifest returned error: %v", err)
	}

	chapter := manifest.Chapters[0]
	if chapter.ID != "003 - Natural" {
		t.Fatalf("unexpected chapter id: %q", chapter.ID)
	}
	if chapter.Title != "003 - Natural" {
		t.Fatalf("unexpected chapter title: %q", chapter.Title)
	}
	if chapter.Number != 3 {
		t.Fatalf("unexpected chapter number: %f", chapter.Number)
	}

	got := []string{
		chapter.Pages[0].FileName,
		chapter.Pages[1].FileName,
		chapter.Pages[2].FileName,
	}
	want := []string{"2.jpg", "3.jpg", "10.jpg"}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("unexpected page order: got %v want %v", got, want)
		}
	}
}

func TestGetReaderManifestSortsChaptersByInferredNumericTokens(t *testing.T) {
	root := t.TempDir()
	mangaDir := filepath.Join(root, "Long Manga")

	chapterNames := []string{
		"1 第1话",
		"10 第10话",
		"100 第100话",
		"101 第101话",
		"2 第2话",
		"第11话",
	}
	for _, chapterName := range chapterNames {
		writeFile(t, filepath.Join(mangaDir, chapterName, "001.jpg"), chapterName)
	}

	manifest, err := GetReaderManifest(root, encodeMangaID("Long Manga"))
	if err != nil {
		t.Fatalf("GetReaderManifest returned error: %v", err)
	}

	got := make([]string, 0, len(manifest.Chapters))
	gotNumbers := make([]float64, 0, len(manifest.Chapters))
	for _, chapter := range manifest.Chapters {
		got = append(got, chapter.Title)
		gotNumbers = append(gotNumbers, chapter.Number)
	}

	want := []string{"1 第1话", "2 第2话", "10 第10话", "第11话", "100 第100话", "101 第101话"}
	wantNumbers := []float64{1, 2, 10, 11, 100, 101}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("unexpected chapter order: got %v want %v", got, want)
		}
		if gotNumbers[index] != wantNumbers[index] {
			t.Fatalf("unexpected chapter numbers: got %v want %v", gotNumbers, wantNumbers)
		}
	}
}

func TestGetReaderManifestArchiveIgnoresUnsupportedAndMetadataEntries(t *testing.T) {
	root := t.TempDir()
	mangaDir := filepath.Join(root, "Filtered Manga")
	archivePath := filepath.Join(mangaDir, "004 - Filter.cbz")

	if err := os.MkdirAll(mangaDir, 0o755); err != nil {
		t.Fatalf("mkdir manga dir: %v", err)
	}

	writeZipArchive(t, archivePath, map[string]string{
		"__MACOSX/._001.jpg": "hidden",
		".DS_Store":          "metadata",
		"notes.txt":          "note",
		"nested/.hidden.jpg": "skip",
		"nested/001.jpg":     "one",
		"002.png":            "two",
	})

	manifest, err := GetReaderManifest(root, encodeMangaID("Filtered Manga"))
	if err != nil {
		t.Fatalf("GetReaderManifest returned error: %v", err)
	}

	pages := manifest.Chapters[0].Pages
	if len(pages) != 2 {
		t.Fatalf("unexpected page count: %d", len(pages))
	}
	got := []string{pages[0].FileName, pages[1].FileName}
	want := []string{"002.png", "nested/001.jpg"}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("unexpected kept entries: got %v want %v", got, want)
		}
	}
}

func TestOpenArchiveAssetValidatesRequests(t *testing.T) {
	root := t.TempDir()
	mangaDir := filepath.Join(root, "Asset Manga")
	archivePath := filepath.Join(mangaDir, "001 - Asset.cbz")

	if err := os.MkdirAll(mangaDir, 0o755); err != nil {
		t.Fatalf("mkdir manga dir: %v", err)
	}

	writeZipArchive(t, archivePath, map[string]string{
		"001.jpg":   "one",
		"notes.txt": "note",
	})

	tests := []struct {
		name         string
		requestURL   string
		wantNotFound bool
	}{
		{
			name:       "invalid base64",
			requestURL: LibraryArchiveAssetPrefix + "bad!/bad!",
		},
		{
			name:       "path traversal archive",
			requestURL: LibraryArchiveAssetPrefix + encodePathToken("../escape.cbz") + "/" + encodePathToken("001.jpg"),
		},
		{
			name:         "missing entry",
			requestURL:   LibraryArchiveAssetPrefix + encodePathToken("Asset Manga/001 - Asset.cbz") + "/" + encodePathToken("missing.jpg"),
			wantNotFound: true,
		},
		{
			name:       "unsupported entry extension",
			requestURL: LibraryArchiveAssetPrefix + encodePathToken("Asset Manga/001 - Asset.cbz") + "/" + encodePathToken("notes.txt"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			reader, _, _, err := OpenArchiveAsset(root, test.requestURL)
			if reader != nil {
				reader.Close()
			}
			if err == nil {
				t.Fatal("expected request to fail")
			}
			if got := os.IsNotExist(err); got != test.wantNotFound {
				t.Fatalf("unexpected not-found flag: got %v want %v (err=%v)", got, test.wantNotFound, err)
			}
		})
	}
}

func TestListLibraryMangaNestedChapterDirectories(t *testing.T) {
	root := t.TempDir()
	mangaDir := filepath.Join(root, "Nested Manga")
	chapterDir := filepath.Join(mangaDir, "默认", "1")

	if err := os.MkdirAll(chapterDir, 0o755); err != nil {
		t.Fatalf("mkdir nested chapter dir: %v", err)
	}

	writeFile(t, filepath.Join(chapterDir, "001.jpg"), "one")
	writeFile(t, filepath.Join(chapterDir, "002.jpg"), "two")

	library, err := ListLibraryManga(root)
	if err != nil {
		t.Fatalf("ListLibraryManga returned error: %v", err)
	}

	if len(library) != 1 {
		t.Fatalf("unexpected library size: %d", len(library))
	}

	item := library[0]
	if item.Title != "Nested Manga" {
		t.Fatalf("unexpected title: %q", item.Title)
	}
	if item.PageCount != 2 {
		t.Fatalf("unexpected page count: %d", item.PageCount)
	}
	if item.ChapterCount != 1 {
		t.Fatalf("unexpected chapter count: %d", item.ChapterCount)
	}

	manifest, err := GetReaderManifest(root, item.ID)
	if err != nil {
		t.Fatalf("GetReaderManifest returned error: %v", err)
	}

	if manifest.TotalPages != 2 {
		t.Fatalf("unexpected total pages: %d", manifest.TotalPages)
	}
	if len(manifest.Chapters) != 1 {
		t.Fatalf("unexpected chapter count in manifest: %d", len(manifest.Chapters))
	}
	if manifest.Chapters[0].Title != "1" {
		t.Fatalf("unexpected chapter title: %q", manifest.Chapters[0].Title)
	}
	if manifest.Chapters[0].Number != 1 {
		t.Fatalf("unexpected chapter number: %f", manifest.Chapters[0].Number)
	}
}

func TestListLibraryMangaNestedChaptersHaveUniqueIDs(t *testing.T) {
	root := t.TempDir()
	mangaDir := filepath.Join(root, "Multi Volume Manga")

	// Two grouping folders, each containing a chapter named "001"
	chapter1Dir := filepath.Join(mangaDir, "Vol 1", "001")
	chapter2Dir := filepath.Join(mangaDir, "Vol 2", "001")

	if err := os.MkdirAll(chapter1Dir, 0o755); err != nil {
		t.Fatalf("mkdir chapter1 dir: %v", err)
	}
	if err := os.MkdirAll(chapter2Dir, 0o755); err != nil {
		t.Fatalf("mkdir chapter2 dir: %v", err)
	}

	writeFile(t, filepath.Join(chapter1Dir, "001.jpg"), "page1")
	writeFile(t, filepath.Join(chapter2Dir, "001.jpg"), "page2")

	library, err := ListLibraryManga(root)
	if err != nil {
		t.Fatalf("ListLibraryManga returned error: %v", err)
	}

	if len(library) != 1 {
		t.Fatalf("unexpected library size: %d", len(library))
	}

	manifest, err := GetReaderManifest(root, library[0].ID)
	if err != nil {
		t.Fatalf("GetReaderManifest returned error: %v", err)
	}

	if len(manifest.Chapters) != 2 {
		t.Fatalf("unexpected chapter count: %d", len(manifest.Chapters))
	}

	// Both chapters have the same directory name but must have unique IDs
	if manifest.Chapters[0].ID == manifest.Chapters[1].ID {
		t.Fatalf("chapters must have unique IDs, got duplicate: %q", manifest.Chapters[0].ID)
	}

	// Both should resolve the title and number from the inner directory name
	for _, ch := range manifest.Chapters {
		if ch.Title != "001" {
			t.Fatalf("unexpected chapter title: %q", ch.Title)
		}
		if ch.Number != 1 {
			t.Fatalf("unexpected chapter number: %f", ch.Number)
		}
	}
}

func TestListLibraryMangaNestedChaptersUniqueIDsWithSidecarsWithoutChapterID(t *testing.T) {
	root := t.TempDir()
	mangaDir := filepath.Join(root, "Sidecar Manga")

	chapter1Dir := filepath.Join(mangaDir, "Vol 1", "001")
	chapter2Dir := filepath.Join(mangaDir, "Vol 2", "001")

	for _, dir := range []string{chapter1Dir, chapter2Dir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
	}

	writeFile(t, filepath.Join(chapter1Dir, "001.jpg"), "page1")
	writeFile(t, filepath.Join(chapter2Dir, "001.jpg"), "page2")

	// Write sidecars that do NOT set chapterID (simulating legacy/partial sidecars)
	for _, dir := range []string{chapter1Dir, chapter2Dir} {
		if err := WriteSidecar(SidecarPath(dir), ChapterSidecar{
			MangaTitle:    "Sidecar Manga",
			ChapterTitle:  "001",
			ChapterNumber: 1,
			// ChapterID intentionally left empty
		}); err != nil {
			t.Fatalf("write sidecar: %v", err)
		}
	}

	library, err := ListLibraryManga(root)
	if err != nil {
		t.Fatalf("ListLibraryManga returned error: %v", err)
	}

	manifest, err := GetReaderManifest(root, library[0].ID)
	if err != nil {
		t.Fatalf("GetReaderManifest returned error: %v", err)
	}

	if len(manifest.Chapters) != 2 {
		t.Fatalf("unexpected chapter count: %d", len(manifest.Chapters))
	}

	if manifest.Chapters[0].ID == manifest.Chapters[1].ID {
		t.Fatalf("chapters must have unique IDs even with sidecars lacking chapterID, got duplicate: %q", manifest.Chapters[0].ID)
	}
}

func TestListLibraryMangaSidecarChapterWithNestedImages(t *testing.T) {
	root := t.TempDir()
	mangaDir := filepath.Join(root, "Sidecar Manga")
	chapterDir := filepath.Join(mangaDir, "001 - Chapter")
	pagesDir := filepath.Join(chapterDir, "pages")

	if err := os.MkdirAll(pagesDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	// Images live in a child folder, not directly in the chapter directory
	writeFile(t, filepath.Join(pagesDir, "001.jpg"), "first")
	writeFile(t, filepath.Join(pagesDir, "002.jpg"), "second")

	// Sidecar references images via relative paths from the chapter dir
	if err := WriteSidecar(SidecarPath(chapterDir), ChapterSidecar{
		SourceURL:         "https://example.com/manga",
		MangaTitle:        "Sidecar Manga",
		ChapterID:         "ch-sidecar-001",
		ChapterNumber:     1,
		ChapterTitle:      "Chapter With Sidecar",
		ExpectedPageCount: 2,
		DownloadedPages:   2,
		Files: []ChapterSidecarFile{
			{PageIndex: 0, FileName: "pages/001.jpg"},
			{PageIndex: 1, FileName: "pages/002.jpg"},
		},
		CompletedAt: "2026-04-25T00:00:00Z",
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

	manifest, err := GetReaderManifest(root, library[0].ID)
	if err != nil {
		t.Fatalf("GetReaderManifest returned error: %v", err)
	}

	if len(manifest.Chapters) != 1 {
		t.Fatalf("unexpected chapter count: %d (chapter should not have been fragmented into child directories)", len(manifest.Chapters))
	}

	chapter := manifest.Chapters[0]

	// Must use sidecar metadata, not "pages" (the child directory name)
	if chapter.ID != "ch-sidecar-001" {
		t.Fatalf("unexpected chapter id: %q (expected sidecar chapterID)", chapter.ID)
	}
	if chapter.Title != "Chapter With Sidecar" {
		t.Fatalf("unexpected chapter title: %q (expected sidecar title)", chapter.Title)
	}
	if chapter.Number != 1 {
		t.Fatalf("unexpected chapter number: %f", chapter.Number)
	}
	if chapter.PageCount != 2 {
		t.Fatalf("unexpected page count: %d", chapter.PageCount)
	}

	// Pages must be served from the filesystem (under chapters parent)
	if len(chapter.Pages) != 2 {
		t.Fatalf("unexpected pages count: %d", len(chapter.Pages))
	}
	for _, p := range chapter.Pages {
		if !strings.HasPrefix(p.SourceURL, LibraryAssetPrefix) {
			t.Fatalf("expected filesystem asset URL, got %q", p.SourceURL)
		}
	}
}

func writeFile(t *testing.T, filePath string, content string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		t.Fatalf("mkdir parent dir: %v", err)
	}
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		t.Fatalf("write file %s: %v", filePath, err)
	}
}

func writeZipArchive(t *testing.T, archivePath string, files map[string]string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(archivePath), 0o755); err != nil {
		t.Fatalf("mkdir archive dir: %v", err)
	}

	file, err := os.Create(archivePath)
	if err != nil {
		t.Fatalf("create archive: %v", err)
	}
	defer file.Close()

	archiveWriter := zip.NewWriter(file)
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		writer, err := archiveWriter.Create(name)
		if err != nil {
			t.Fatalf("create archive entry %s: %v", name, err)
		}
		if _, err := io.WriteString(writer, files[name]); err != nil {
			t.Fatalf("write archive entry %s: %v", name, err)
		}
	}
	if err := archiveWriter.Close(); err != nil {
		t.Fatalf("close archive writer: %v", err)
	}
}

func sameResolvedPath(left string, right string) bool {
	resolvedLeft, err := filepath.EvalSymlinks(left)
	if err != nil {
		resolvedLeft = left
	}
	resolvedRight, err := filepath.EvalSymlinks(right)
	if err != nil {
		resolvedRight = right
	}
	return resolvedLeft == resolvedRight
}
