package main

import (
	"archive/zip"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/sakagamijun/rawmanga-download-go/internal/contracts"
	"github.com/sakagamijun/rawmanga-download-go/internal/download"
	"github.com/sakagamijun/rawmanga-download-go/internal/klz9"
	"github.com/sakagamijun/rawmanga-download-go/internal/settings"
	"github.com/sakagamijun/rawmanga-download-go/internal/store"
)

func TestAssetHandlerStreamsArchiveAssets(t *testing.T) {
	outputRoot := t.TempDir()
	mangaDir := filepath.Join(outputRoot, "Reader Manga")
	archivePath := filepath.Join(mangaDir, "001 - Chapter.cbz")

	if err := os.MkdirAll(mangaDir, 0o755); err != nil {
		t.Fatalf("mkdir manga dir: %v", err)
	}
	writeZipArchiveForAppTest(t, archivePath, map[string]string{
		"001.png": "png-bytes",
	})

	app, cleanup := newAssetTestApp(t, outputRoot)
	defer cleanup()

	requestURL, err := download.ArchiveAssetURL(outputRoot, archivePath, "001.png")
	if err != nil {
		t.Fatalf("build archive asset url: %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, requestURL, nil)
	recorder := httptest.NewRecorder()
	app.AssetHandler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", recorder.Code)
	}
	if got := recorder.Header().Get("Content-Type"); got != "image/png" {
		t.Fatalf("unexpected content type: %q", got)
	}
	if got := recorder.Header().Get("Content-Length"); got != "9" {
		t.Fatalf("unexpected content length: %q", got)
	}
	if got := recorder.Body.String(); got != "png-bytes" {
		t.Fatalf("unexpected body: %q", got)
	}
}

func TestAssetHandlerArchiveNotFoundScenarios(t *testing.T) {
	outputRoot := t.TempDir()
	mangaDir := filepath.Join(outputRoot, "Reader Manga")
	archivePath := filepath.Join(mangaDir, "001 - Chapter.cbz")

	if err := os.MkdirAll(mangaDir, 0o755); err != nil {
		t.Fatalf("mkdir manga dir: %v", err)
	}
	writeZipArchiveForAppTest(t, archivePath, map[string]string{
		"001.jpg": "one",
	})

	app, cleanup := newAssetTestApp(t, outputRoot)
	defer cleanup()

	tests := []struct {
		name       string
		requestURL string
		wantStatus int
	}{
		{
			name:       "missing archive",
			requestURL: download.LibraryArchiveAssetPrefix + encodePathTokenForAppTest("Reader Manga/missing.cbz") + "/" + encodePathTokenForAppTest("001.jpg"),
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "missing entry",
			requestURL: download.LibraryArchiveAssetPrefix + encodePathTokenForAppTest("Reader Manga/001 - Chapter.cbz") + "/" + encodePathTokenForAppTest("missing.jpg"),
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "invalid decode",
			requestURL: download.LibraryArchiveAssetPrefix + "bad!/bad!",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "escaping entry path",
			requestURL: download.LibraryArchiveAssetPrefix + encodePathTokenForAppTest("Reader Manga/001 - Chapter.cbz") + "/" + encodePathTokenForAppTest("../001.jpg"),
			wantStatus: http.StatusForbidden,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, test.requestURL, nil)
			recorder := httptest.NewRecorder()
			app.AssetHandler().ServeHTTP(recorder, request)

			if recorder.Code != test.wantStatus {
				t.Fatalf("unexpected status: got %d want %d", recorder.Code, test.wantStatus)
			}
		})
	}
}

func newAssetTestApp(t *testing.T, outputRoot string) (*App, func()) {
	t.Helper()

	storeValue, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open store: %v", err)
	}

	settingsService, err := settings.NewService(storeValue)
	if err != nil {
		t.Fatalf("create settings service: %v", err)
	}
	if _, err := settingsService.Update(contracts.AppSettings{OutputRoot: outputRoot}); err != nil {
		t.Fatalf("update settings: %v", err)
	}

	return &App{
			store:     storeValue,
			settings:  settingsService,
			klz9:      &klz9.Service{},
			downloads: &download.Manager{},
		}, func() {
			_ = storeValue.Close()
		}
}

func encodePathTokenForAppTest(value string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(value))
}

func writeZipArchiveForAppTest(t *testing.T, archivePath string, files map[string]string) {
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
