package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sakagamijun/rawmanga-download-go/internal/contracts"
	"github.com/sakagamijun/rawmanga-download-go/internal/download"
	"github.com/sakagamijun/rawmanga-download-go/internal/klz9"
	"github.com/sakagamijun/rawmanga-download-go/internal/settings"
	"github.com/sakagamijun/rawmanga-download-go/internal/store"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx       context.Context
	bootErr   error
	store     *store.SQLiteStore
	settings  *settings.Service
	klz9      *klz9.Service
	downloads *download.Manager
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.bootErr = a.bootstrap()
}

func (a *App) bootstrap() error {
	dataDir, err := os.UserConfigDir()
	if err != nil {
		return contracts.ContractError{
			Code:    contracts.ErrCodeBootstrapFailure,
			Message: fmt.Sprintf("locate user config dir: %v", err),
		}
	}

	appDataDir := filepath.Join(dataDir, "klz9-downloader")
	storeValue, err := store.Open(appDataDir)
	if err != nil {
		return contracts.ContractError{
			Code:    contracts.ErrCodeBootstrapFailure,
			Message: err.Error(),
		}
	}

	settingsService, err := settings.NewService(storeValue)
	if err != nil {
		return contracts.ContractError{
			Code:    contracts.ErrCodeBootstrapFailure,
			Message: err.Error(),
		}
	}

	klz9Service, err := klz9.NewService(storeValue, settingsService.Get().RequestTimeoutSec)
	if err != nil {
		return contracts.ContractError{
			Code:    contracts.ErrCodeBootstrapFailure,
			Message: err.Error(),
		}
	}

	downloadManager := download.NewManager(storeValue, settingsService, klz9Service)
	downloadManager.SetEmitter(a.emit)

	a.store = storeValue
	a.settings = settingsService
	a.klz9 = klz9Service
	a.downloads = downloadManager

	a.emit(contracts.EventSettingsUpdated, settingsService.Get())

	return nil
}

func (a *App) ResolveManga(inputURL string) (contracts.ParsedMangaResult, error) {
	if err := a.ensureReady(); err != nil {
		return contracts.ParsedMangaResult{}, err
	}

	resolved, err := a.klz9.ResolveManga(a.ctx, inputURL)
	if err != nil {
		return contracts.ParsedMangaResult{}, err
	}

	localStates, summary, err := a.reconcileResolvedManga(resolved)
	if err != nil {
		return contracts.ParsedMangaResult{}, err
	}

	stateByChapterID := make(map[string]contracts.LocalChapterState, len(localStates))
	for _, state := range localStates {
		stateByChapterID[state.ChapterID] = state
	}

	chapters := make([]contracts.ChapterItem, 0, len(resolved.Chapters))
	for _, chapter := range resolved.Chapters {
		localState := stateByChapterID[chapter.ID]
		chapters = append(chapters, contracts.ChapterItem{
			ID:          chapter.ID,
			Number:      chapter.Number,
			Title:       chapter.Title,
			ReleaseDate: chapter.ReleaseDate,
			PageCount:   len(chapter.Pages),
			LocalStatus: localState.Status,
			LocalPath:   localState.LocalPath,
			Selected:    false,
		})
	}

	result := contracts.ParsedMangaResult{
		SourceURL:        resolved.SourceURL,
		Slug:             resolved.Slug,
		Title:            resolved.Title,
		CoverURL:         resolved.CoverURL,
		Chapters:         chapters,
		LocalSummary:     summary,
		ProfileCacheHit:  resolved.ProfileCacheHit,
		AlgorithmProfile: resolved.Profile.Site + ":" + resolved.Profile.BundleHash,
	}

	a.emit(contracts.EventLibraryReconciled, result)

	return result, nil
}

func (a *App) QueueChapters(request contracts.QueueDownloadRequest) (contracts.DownloadJob, error) {
	if err := a.ensureReady(); err != nil {
		return contracts.DownloadJob{}, err
	}

	return a.downloads.QueueChapters(a.ctx, request)
}

func (a *App) ListDownloadJobs() ([]contracts.DownloadJob, error) {
	if err := a.ensureReady(); err != nil {
		return nil, err
	}

	return a.downloads.ListDownloadJobs()
}

func (a *App) PauseJob(jobID string) error {
	if err := a.ensureReady(); err != nil {
		return err
	}

	return a.downloads.PauseJob(jobID)
}

func (a *App) ResumeJob(jobID string) error {
	if err := a.ensureReady(); err != nil {
		return err
	}

	return a.downloads.ResumeJob(jobID)
}

func (a *App) RetryFailed(jobID string) error {
	if err := a.ensureReady(); err != nil {
		return err
	}

	return a.downloads.RetryFailed(a.ctx, jobID)
}

func (a *App) ScanLocalState(sourceURL string) ([]contracts.LocalChapterState, error) {
	if err := a.ensureReady(); err != nil {
		return nil, err
	}

	resolved, err := a.klz9.ResolveManga(a.ctx, sourceURL)
	if err != nil {
		return nil, err
	}

	states, _, err := a.reconcileResolvedManga(resolved)
	return states, err
}

func (a *App) GetSettings() (contracts.AppSettings, error) {
	if err := a.ensureReady(); err != nil {
		return contracts.AppSettings{}, err
	}

	return a.settings.Get(), nil
}

func (a *App) UpdateSettings(input contracts.AppSettings) (contracts.AppSettings, error) {
	if err := a.ensureReady(); err != nil {
		return contracts.AppSettings{}, err
	}

	updated, err := a.settings.Update(input)
	if err != nil {
		return contracts.AppSettings{}, err
	}

	a.klz9.SetTimeout(updated.RequestTimeoutSec)
	a.emit(contracts.EventSettingsUpdated, updated)

	return updated, nil
}

func (a *App) ListLibraryManga() ([]contracts.LibraryManga, error) {
	if err := a.ensureReady(); err != nil {
		return nil, err
	}

	return download.ListLibraryManga(a.settings.Get().OutputRoot)
}

func (a *App) GetReaderManifest(mangaID string) (contracts.ReaderManifest, error) {
	if err := a.ensureReady(); err != nil {
		return contracts.ReaderManifest{}, err
	}

	return download.GetReaderManifest(a.settings.Get().OutputRoot, mangaID)
}

func (a *App) GetReaderProgress(mangaID string) (contracts.ReaderProgress, error) {
	if err := a.ensureReady(); err != nil {
		return contracts.ReaderProgress{}, err
	}

	progress, found, err := a.store.GetReaderProgress(mangaID)
	if err != nil {
		return contracts.ReaderProgress{}, err
	}
	if !found {
		return contracts.ReaderProgress{MangaID: mangaID}, nil
	}

	return progress, nil
}

func (a *App) UpdateReaderProgress(input contracts.ReaderProgress) (contracts.ReaderProgress, error) {
	if err := a.ensureReady(); err != nil {
		return contracts.ReaderProgress{}, err
	}

	if input.Page < 1 {
		input.Page = 1
	}
	input.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	if err := a.store.SaveReaderProgress(input); err != nil {
		return contracts.ReaderProgress{}, err
	}

	return input, nil
}

func (a *App) AssetHandler() http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if !download.IsLibraryAssetRequest(request.URL.Path) {
			http.NotFound(writer, request)
			return
		}

		if err := a.ensureReady(); err != nil {
			http.Error(writer, err.Error(), http.StatusServiceUnavailable)
			return
		}

		outputRoot := a.settings.Get().OutputRoot
		if strings.HasPrefix(request.URL.Path, download.LibraryArchiveAssetPrefix) {
			reader, contentType, contentLength, err := download.OpenArchiveAsset(outputRoot, request.URL.Path)
			if err != nil {
				if os.IsNotExist(err) {
					http.NotFound(writer, request)
					return
				}
				http.Error(writer, err.Error(), http.StatusForbidden)
				return
			}
			defer reader.Close()

			if contentType != "" {
				writer.Header().Set("Content-Type", contentType)
			}
			if contentLength >= 0 {
				writer.Header().Set("Content-Length", fmt.Sprintf("%d", contentLength))
			}

			_, _ = io.Copy(writer, reader)
			return
		}

		targetPath, err := download.ResolveLibraryAssetPath(outputRoot, request.URL.Path)
		if err != nil {
			if os.IsNotExist(err) {
				http.NotFound(writer, request)
				return
			}
			http.Error(writer, err.Error(), http.StatusForbidden)
			return
		}

		http.ServeFile(writer, request, targetPath)
	})
}

func (a *App) emit(event string, payload any) {
	if a.ctx == nil {
		return
	}

	runtime.EventsEmit(a.ctx, event, payload)
}

func (a *App) ensureReady() error {
	if a.bootErr != nil {
		return a.bootErr
	}

	if a.store == nil || a.settings == nil || a.klz9 == nil || a.downloads == nil {
		return contracts.ContractError{
			Code:    contracts.ErrCodeBootstrapFailure,
			Message: "application services are not initialized",
		}
	}

	return nil
}

func (a *App) reconcileResolvedManga(resolved klz9.ResolvedManga) ([]contracts.LocalChapterState, contracts.LocalSummary, error) {
	settingsValue := a.settings.Get()
	mangaDir := download.MangaDirectory(settingsValue.OutputRoot, resolved.Title)

	var (
		states  []contracts.LocalChapterState
		summary contracts.LocalSummary
	)

	for _, chapter := range resolved.Chapters {
		chapterDir := download.ChapterDirectory(mangaDir, chapter.Number, chapter.Title)
		localCount, dirExists, err := countChapterImages(chapterDir)
		if err != nil {
			return nil, contracts.LocalSummary{}, err
		}

		sidecar, sidecarFound, err := download.ReadSidecar(download.SidecarPath(chapterDir))
		if err != nil {
			return nil, contracts.LocalSummary{}, err
		}

		expectedPages := len(chapter.Pages)
		if sidecarFound && sidecar.ExpectedPageCount > 0 {
			expectedPages = sidecar.ExpectedPageCount
		}

		status := contracts.LocalChapterStatusNotDownloaded
		switch {
		case localCount == 0 && !dirExists && !sidecarFound:
			status = contracts.LocalChapterStatusNotDownloaded
		case localCount > 0 && expectedPages > 0 && localCount >= expectedPages:
			status = contracts.LocalChapterStatusComplete
		case sidecarFound && sidecar.CompletedAt != "" && localCount < expectedPages:
			status = contracts.LocalChapterStatusMissing
		default:
			status = contracts.LocalChapterStatusPartial
		}

		localPath := ""
		if dirExists || sidecarFound {
			localPath = chapterDir
		}

		state := contracts.LocalChapterState{
			ChapterID:         chapter.ID,
			ChapterNumber:     chapter.Number,
			Title:             chapter.Title,
			Status:            status,
			LocalPath:         localPath,
			LocalPageCount:    localCount,
			ExpectedPageCount: expectedPages,
		}
		states = append(states, state)

		if err := a.store.SaveLocalChapterState(resolved.Slug, chapter.ID, resolved.SourceURL, state); err != nil {
			return nil, contracts.LocalSummary{}, err
		}

		switch status {
		case contracts.LocalChapterStatusNotDownloaded:
			summary.NotDownloaded++
		case contracts.LocalChapterStatusPartial:
			summary.Partial++
		case contracts.LocalChapterStatusComplete:
			summary.Complete++
		case contracts.LocalChapterStatusMissing:
			summary.Missing++
		}
	}

	return states, summary, nil
}

func countChapterImages(chapterDir string) (int, bool, error) {
	entries, err := os.ReadDir(chapterDir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, false, nil
		}

		return 0, false, fmt.Errorf("read chapter directory: %w", err)
	}

	count := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if strings.HasPrefix(name, ".") || strings.HasSuffix(name, ".part") {
			continue
		}

		switch strings.ToLower(filepath.Ext(name)) {
		case ".jpg", ".jpeg", ".png", ".webp", ".gif", ".avif":
			count++
		}
	}

	return count, true, nil
}
