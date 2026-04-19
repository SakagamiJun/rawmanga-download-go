package download

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sakagamijun/rawmanga-download-go/internal/contracts"
	"github.com/sakagamijun/rawmanga-download-go/internal/klz9"
	"github.com/sakagamijun/rawmanga-download-go/internal/settings"
	"github.com/sakagamijun/rawmanga-download-go/internal/store"
)

type Manager struct {
	store    *store.SQLiteStore
	settings *settings.Service
	klz9     *klz9.Service
	client   *http.Client

	mu      sync.Mutex
	jobs    map[string]*jobState
	emitter func(string, any)
}

type jobState struct {
	mu         sync.Mutex
	cond       *sync.Cond
	job        contracts.DownloadJob
	request    contracts.QueueDownloadRequest
	profile    contracts.SiteProfile
	chapters   []klz9.ResolvedChapter
	payloadRaw string
	paused     bool
	running    bool
}

func NewManager(store *store.SQLiteStore, settings *settings.Service, klz9Service *klz9.Service) *Manager {
	return &Manager{
		store:    store,
		settings: settings,
		klz9:     klz9Service,
		client:   &http.Client{Timeout: 30 * time.Second},
		jobs:     make(map[string]*jobState),
		emitter:  func(string, any) {},
	}
}

func (m *Manager) SetEmitter(emitter func(string, any)) {
	if emitter == nil {
		m.emitter = func(string, any) {}
		return
	}

	m.emitter = emitter
}

func (m *Manager) QueueChapters(ctx context.Context, request contracts.QueueDownloadRequest) (contracts.DownloadJob, error) {
	resolved, err := m.klz9.ResolveManga(ctx, request.SourceURL)
	if err != nil {
		return contracts.DownloadJob{}, err
	}

	selection := make(map[string]struct{}, len(request.ChapterIDs))
	for _, chapterID := range request.ChapterIDs {
		selection[chapterID] = struct{}{}
	}

	var selected []klz9.ResolvedChapter
	for _, chapter := range resolved.Chapters {
		if _, ok := selection[chapter.ID]; ok {
			selected = append(selected, chapter)
		}
	}

	if len(selected) == 0 {
		return contracts.DownloadJob{}, contracts.ContractError{
			Code:    contracts.ErrCodeChapterNotFound,
			Message: "no matching chapters selected",
		}
	}

	sort.SliceStable(selected, func(i, j int) bool {
		return selected[i].Number < selected[j].Number
	})

	settingsValue := m.settings.Get()
	m.client.Timeout = time.Duration(settingsValue.RequestTimeoutSec) * time.Second

	if request.OutputRoot == "" {
		request.OutputRoot = settingsValue.OutputRoot
	}
	if request.MangaSlug == "" {
		request.MangaSlug = resolved.Slug
	}
	if request.Title == "" {
		request.Title = resolved.Title
	}

	now := time.Now().UTC().Format(time.RFC3339)
	job := contracts.DownloadJob{
		JobID:              "job_" + uuid.NewString(),
		MangaSlug:          request.MangaSlug,
		MangaTitle:         request.Title,
		SourceURL:          request.SourceURL,
		Status:             contracts.JobStatusQueued,
		QueuedChapters:     len(selected),
		CompletedChapters:  0,
		FailedChapters:     0,
		CreatedAt:          now,
		UpdatedAt:          now,
		LastError:          "",
		MaxConcurrentPages: settingsValue.MaxConcurrentDownloads,
	}

	payload, err := json.Marshal(request)
	if err != nil {
		return contracts.DownloadJob{}, fmt.Errorf("encode queue request: %w", err)
	}

	state := &jobState{
		job:        job,
		request:    request,
		profile:    resolved.Profile,
		chapters:   selected,
		payloadRaw: string(payload),
	}
	state.cond = sync.NewCond(&state.mu)

	if err := m.store.SaveDownloadJob(job, state.payloadRaw); err != nil {
		return contracts.DownloadJob{}, err
	}

	m.mu.Lock()
	m.jobs[job.JobID] = state
	m.mu.Unlock()

	m.emitter(contracts.EventDownloadJob, job)

	go m.runJob(state)

	return job, nil
}

func (m *Manager) ListDownloadJobs() ([]contracts.DownloadJob, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	jobs := make([]contracts.DownloadJob, 0, len(m.jobs))
	for _, state := range m.jobs {
		state.mu.Lock()
		jobCopy := state.job
		state.mu.Unlock()

		if jobCopy.Status == contracts.JobStatusCompleted || jobCopy.Status == contracts.JobStatusCanceled || jobCopy.Status == contracts.JobStatusSkipped {
			continue
		}

		jobs = append(jobs, jobCopy)
	}

	sort.SliceStable(jobs, func(i, j int) bool {
		return jobs[i].UpdatedAt > jobs[j].UpdatedAt
	})

	return jobs, nil
}

func (m *Manager) PauseJob(jobID string) error {
	state, err := m.lookupJob(jobID)
	if err != nil {
		return err
	}

	state.mu.Lock()
	state.paused = true
	state.job.Status = contracts.JobStatusPaused
	state.job.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	state.mu.Unlock()

	return m.persistAndEmitJob(state)
}

func (m *Manager) ResumeJob(jobID string) error {
	state, err := m.lookupJob(jobID)
	if err != nil {
		return err
	}

	state.mu.Lock()
	state.paused = false
	state.job.Status = contracts.JobStatusRunning
	state.job.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	state.cond.Broadcast()
	state.mu.Unlock()

	return m.persistAndEmitJob(state)
}

func (m *Manager) RetryFailed(ctx context.Context, jobID string) error {
	state, err := m.lookupJob(jobID)
	if err != nil {
		return err
	}

	state.mu.Lock()
	if state.running {
		state.mu.Unlock()
		return contracts.ContractError{
			Code:    contracts.ErrCodeDownloadFailed,
			Message: "cannot retry a running job",
		}
	}

	state.job.Status = contracts.JobStatusQueued
	state.job.CompletedChapters = 0
	state.job.FailedChapters = 0
	state.job.LastError = ""
	state.job.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	state.paused = false
	state.mu.Unlock()

	if err := m.persistAndEmitJob(state); err != nil {
		return err
	}

	go m.runJob(state)

	return nil
}

func (m *Manager) lookupJob(jobID string) (*jobState, error) {
	m.mu.Lock()
	state, ok := m.jobs[jobID]
	m.mu.Unlock()

	if ok {
		return state, nil
	}

	return nil, contracts.ContractError{
		Code:    contracts.ErrCodeJobNotFound,
		Message: jobID,
	}
}

func (m *Manager) runJob(state *jobState) {
	state.mu.Lock()
	state.running = true
	state.paused = false
	state.job.Status = contracts.JobStatusRunning
	state.job.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	state.mu.Unlock()

	_ = m.persistAndEmitJob(state)

	for _, chapter := range state.chapters {
		state.waitIfPaused()

		fullChapter, err := m.klz9.FetchChapterByID(context.Background(), state.profile, chapter.ID)
		if err != nil {
			m.failChapter(state, chapter, err)
			continue
		}

		if err := m.downloadChapter(state, fullChapter); err != nil {
			m.failChapter(state, fullChapter, err)
			continue
		}

		state.mu.Lock()
		state.job.CompletedChapters++
		state.job.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
		state.mu.Unlock()
		_ = m.persistAndEmitJob(state)
	}

	state.mu.Lock()
	state.running = false
	if state.job.FailedChapters > 0 {
		state.job.Status = contracts.JobStatusFailed
	} else {
		state.job.Status = contracts.JobStatusCompleted
	}
	state.job.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	state.mu.Unlock()

	_ = m.persistAndEmitJob(state)
}

func (m *Manager) failChapter(state *jobState, chapter klz9.ResolvedChapter, err error) {
	state.mu.Lock()
	state.job.FailedChapters++
	state.job.LastError = err.Error()
	state.job.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	state.mu.Unlock()

	event := contracts.DownloadProgressEvent{
		JobID:        state.job.JobID,
		ChapterID:    chapter.ID,
		ChapterTitle: chapter.Title,
		Status:       contracts.JobStatusFailed,
		Error:        err.Error(),
	}

	_ = m.persistAndEmitJob(state)
	m.emitter(contracts.EventDownloadChapter, event)
}

func (m *Manager) downloadChapter(state *jobState, chapter klz9.ResolvedChapter) error {
	settingsValue := m.settings.Get()
	outputRoot := state.request.OutputRoot
	if outputRoot == "" {
		outputRoot = settingsValue.OutputRoot
	}

	mangaDir := MangaDirectory(outputRoot, state.job.MangaTitle)
	chapterDir := ChapterDirectory(mangaDir, chapter.Number, chapter.Title)
	if err := os.MkdirAll(chapterDir, 0o755); err != nil {
		return fmt.Errorf("create chapter directory: %w", err)
	}

	sidecar := ChapterSidecar{
		SourceURL:         state.request.SourceURL,
		MangaSlug:         state.job.MangaSlug,
		MangaTitle:        state.job.MangaTitle,
		ChapterID:         chapter.ID,
		ChapterNumber:     chapter.Number,
		ChapterTitle:      chapter.Title,
		BundleHash:        state.profile.BundleHash,
		ExpectedPageCount: len(chapter.Pages),
		DownloadedPages:   0,
	}

	if err := WriteSidecar(SidecarPath(chapterDir), sidecar); err != nil {
		return err
	}

	m.emitter(contracts.EventDownloadChapter, contracts.DownloadProgressEvent{
		JobID:        state.job.JobID,
		ChapterID:    chapter.ID,
		ChapterTitle: chapter.Title,
		Status:       contracts.JobStatusRunning,
		TotalPages:   len(chapter.Pages),
		LocalStatus:  string(contracts.LocalChapterStatusPartial),
	})

	type result struct {
		index       int
		fileName    string
		pageURL     string
		bytesPerSec int64
		size        int64
		err         error
	}

	sem := make(chan struct{}, max(1, settingsValue.MaxConcurrentDownloads))
	results := make(chan result, len(chapter.Pages))
	var group sync.WaitGroup

	for index, pageURL := range chapter.Pages {
		index := index
		pageURL := pageURL

		group.Add(1)
		go func() {
			defer group.Done()

			state.waitIfPaused()
			sem <- struct{}{}
			defer func() { <-sem }()
			state.waitIfPaused()

			extension := imageExtension(pageURL)
			fileName := ImageFilename(index, extension)
			finalPath := filepath.Join(chapterDir, fileName)
			tmpPath := finalPath + ".part"

			if info, err := os.Stat(finalPath); err == nil && info.Size() > 0 {
				_ = m.store.SaveDownloadedFile(state.job.MangaSlug, chapter.ID, index, finalPath, pageURL, "done", info.Size())
				results <- result{
					index:       index,
					fileName:    fileName,
					pageURL:     pageURL,
					size:        info.Size(),
					bytesPerSec: 0,
				}
				return
			}

			startedAt := time.Now()
			size, err := m.downloadPageWithRetry(pageURL, tmpPath, finalPath, settingsValue.RetryCount)
			if err != nil {
				_ = os.Remove(tmpPath)
				_ = m.store.SaveDownloadedFile(state.job.MangaSlug, chapter.ID, index, finalPath, pageURL, "failed", 0)
				results <- result{
					index:    index,
					fileName: fileName,
					pageURL:  pageURL,
					err:      err,
				}
				return
			}

			duration := time.Since(startedAt)
			bytesPerSecond := int64(0)
			if duration > 0 {
				bytesPerSecond = int64(float64(size) / duration.Seconds())
			}

			_ = m.store.SaveDownloadedFile(state.job.MangaSlug, chapter.ID, index, finalPath, pageURL, "done", size)
			results <- result{
				index:       index,
				fileName:    fileName,
				pageURL:     pageURL,
				bytesPerSec: bytesPerSecond,
				size:        size,
			}
		}()
	}

	go func() {
		group.Wait()
		close(results)
	}()

	downloadedPages := 0
	failedPages := 0
	files := make([]ChapterSidecarFile, 0, len(chapter.Pages))
	for resultItem := range results {
		if resultItem.err != nil {
			failedPages++
			m.emitter(contracts.EventDownloadPage, contracts.DownloadProgressEvent{
				JobID:           state.job.JobID,
				ChapterID:       chapter.ID,
				ChapterTitle:    chapter.Title,
				PageIndex:       resultItem.index,
				Status:          contracts.JobStatusFailed,
				DownloadedPages: downloadedPages,
				TotalPages:      len(chapter.Pages),
				CurrentFile:     resultItem.fileName,
				Error:           resultItem.err.Error(),
				LocalStatus:     string(contracts.LocalChapterStatusPartial),
			})
			continue
		}

		downloadedPages++
		files = append(files, ChapterSidecarFile{
			PageIndex: resultItem.index,
			FileName:  resultItem.fileName,
			URL:       resultItem.pageURL,
		})
		m.emitter(contracts.EventDownloadPage, contracts.DownloadProgressEvent{
			JobID:           state.job.JobID,
			ChapterID:       chapter.ID,
			ChapterTitle:    chapter.Title,
			PageIndex:       resultItem.index,
			Status:          contracts.JobStatusRunning,
			DownloadedPages: downloadedPages,
			TotalPages:      len(chapter.Pages),
			CurrentFile:     resultItem.fileName,
			BytesPerSec:     resultItem.bytesPerSec,
			LocalStatus:     string(contracts.LocalChapterStatusPartial),
		})
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].PageIndex < files[j].PageIndex
	})

	localStatus := contracts.LocalChapterStatusComplete
	completedAt := time.Now().UTC().Format(time.RFC3339)
	if failedPages > 0 {
		localStatus = contracts.LocalChapterStatusPartial
		completedAt = ""
	}

	sidecar.DownloadedPages = downloadedPages
	sidecar.Files = files
	sidecar.CompletedAt = completedAt
	if err := WriteSidecar(SidecarPath(chapterDir), sidecar); err != nil {
		return err
	}

	statePayload := contracts.LocalChapterState{
		ChapterID:         chapter.ID,
		ChapterNumber:     chapter.Number,
		Title:             chapter.Title,
		Status:            localStatus,
		LocalPath:         chapterDir,
		LocalPageCount:    downloadedPages,
		ExpectedPageCount: len(chapter.Pages),
	}
	if err := m.store.SaveLocalChapterState(state.job.MangaSlug, chapter.ID, state.request.SourceURL, statePayload); err != nil {
		return err
	}

	if failedPages > 0 {
		return fmt.Errorf("chapter %s finished with %d failed pages", chapter.ID, failedPages)
	}

	m.emitter(contracts.EventDownloadChapter, contracts.DownloadProgressEvent{
		JobID:           state.job.JobID,
		ChapterID:       chapter.ID,
		ChapterTitle:    chapter.Title,
		Status:          contracts.JobStatusCompleted,
		DownloadedPages: downloadedPages,
		TotalPages:      len(chapter.Pages),
		LocalStatus:     string(localStatus),
		CompletedAt:     completedAt,
	})

	return nil
}

func (m *Manager) downloadPageWithRetry(pageURL string, tmpPath string, finalPath string, retryCount int) (int64, error) {
	attempts := max(1, retryCount+1)
	var lastErr error

	for attempt := 1; attempt <= attempts; attempt++ {
		size, err := m.downloadPage(pageURL, tmpPath, finalPath)
		if err == nil {
			return size, nil
		}

		lastErr = err
		_ = os.Remove(tmpPath)
	}

	return 0, fmt.Errorf("download page after %d attempts: %w", attempts, lastErr)
}

func (m *Manager) downloadPage(pageURL string, tmpPath string, finalPath string) (int64, error) {
	request, err := http.NewRequest(http.MethodGet, pageURL, nil)
	if err != nil {
		return 0, fmt.Errorf("create page request: %w", err)
	}

	request.Header.Set("User-Agent", "KLZ9Downloader/0.1")
	response, err := m.client.Do(request)
	if err != nil {
		return 0, fmt.Errorf("download page: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return 0, fmt.Errorf("download page returned status %d", response.StatusCode)
	}

	_ = os.Remove(tmpPath)
	file, err := os.Create(tmpPath)
	if err != nil {
		return 0, fmt.Errorf("create temp file: %w", err)
	}

	size, copyErr := io.Copy(file, response.Body)
	closeErr := file.Close()
	if copyErr != nil {
		return 0, fmt.Errorf("write temp file: %w", copyErr)
	}
	if closeErr != nil {
		return 0, fmt.Errorf("close temp file: %w", closeErr)
	}

	if err := os.Rename(tmpPath, finalPath); err != nil {
		return 0, fmt.Errorf("rename temp file: %w", err)
	}

	return size, nil
}

func (m *Manager) persistAndEmitJob(state *jobState) error {
	state.mu.Lock()
	jobCopy := state.job
	payloadRaw := state.payloadRaw
	state.mu.Unlock()

	if err := m.store.SaveDownloadJob(jobCopy, payloadRaw); err != nil {
		return err
	}

	m.emitter(contracts.EventDownloadJob, jobCopy)
	return nil
}

func imageExtension(pageURL string) string {
	parsedURL, err := url.Parse(pageURL)
	if err != nil {
		return ".jpg"
	}

	extension := strings.ToLower(path.Ext(parsedURL.Path))
	switch extension {
	case ".jpg", ".jpeg", ".png", ".webp", ".gif", ".avif":
		return extension
	default:
		return ".jpg"
	}
}

func max(a int, b int) int {
	if a > b {
		return a
	}
	return b
}

func (j *jobState) waitIfPaused() {
	j.mu.Lock()
	defer j.mu.Unlock()

	for j.paused {
		j.cond.Wait()
	}
}
