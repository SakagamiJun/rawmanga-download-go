package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/sakagamijun/rawmanga-download-go/internal/contracts"
)

type SQLiteStore struct {
	db      *sql.DB
	dataDir string
}

func Open(dataDir string) (*SQLiteStore, error) {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}

	dbPath := filepath.Join(dataDir, "app.db")
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	db.SetMaxOpenConns(1)

	store := &SQLiteStore{
		db:      db,
		dataDir: dataDir,
	}

	if err := store.init(); err != nil {
		_ = db.Close()
		return nil, err
	}

	return store, nil
}

func (s *SQLiteStore) Close() error {
	if s == nil || s.db == nil {
		return nil
	}

	return s.db.Close()
}

func (s *SQLiteStore) DataDir() string {
	return s.dataDir
}

func (s *SQLiteStore) init() error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS site_profiles (
			site TEXT NOT NULL,
			bundle_hash TEXT PRIMARY KEY,
			bundle_url TEXT NOT NULL,
			profile_json TEXT NOT NULL,
			extracted_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS download_jobs (
			job_id TEXT PRIMARY KEY,
			manga_slug TEXT NOT NULL,
			manga_title TEXT NOT NULL,
			source_url TEXT NOT NULL,
			status TEXT NOT NULL,
			queued_chapters INTEGER NOT NULL,
			completed_chapters INTEGER NOT NULL,
			failed_chapters INTEGER NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			last_error TEXT NOT NULL,
			max_concurrent_pages INTEGER NOT NULL,
			payload_json TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS chapter_states (
			manga_slug TEXT NOT NULL,
			chapter_id TEXT NOT NULL,
			source_url TEXT NOT NULL,
			payload_json TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			PRIMARY KEY (manga_slug, chapter_id)
		);`,
		`CREATE TABLE IF NOT EXISTS downloaded_files (
			manga_slug TEXT NOT NULL,
			chapter_id TEXT NOT NULL,
			page_index INTEGER NOT NULL,
			file_path TEXT NOT NULL,
			page_url TEXT NOT NULL,
			status TEXT NOT NULL,
			size_bytes INTEGER NOT NULL,
			updated_at TEXT NOT NULL,
			PRIMARY KEY (manga_slug, chapter_id, page_index)
		);`,
		`CREATE TABLE IF NOT EXISTS reader_progress (
			manga_id TEXT PRIMARY KEY,
			chapter_id TEXT NOT NULL,
			page INTEGER NOT NULL,
			updated_at TEXT NOT NULL
		);`,
	}

	for _, statement := range statements {
		if _, err := s.db.Exec(statement); err != nil {
			return fmt.Errorf("init schema: %w", err)
		}
	}

	return nil
}

func (s *SQLiteStore) GetSettings() (contracts.AppSettings, bool, error) {
	const query = `SELECT value FROM settings WHERE key = 'app'`

	var raw string
	if err := s.db.QueryRow(query).Scan(&raw); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return contracts.AppSettings{}, false, nil
		}

		return contracts.AppSettings{}, false, fmt.Errorf("select settings: %w", err)
	}

	var settings contracts.AppSettings
	if err := json.Unmarshal([]byte(raw), &settings); err != nil {
		return contracts.AppSettings{}, false, fmt.Errorf("decode settings: %w", err)
	}

	return settings, true, nil
}

func (s *SQLiteStore) SaveSettings(settings contracts.AppSettings) error {
	raw, err := json.Marshal(settings)
	if err != nil {
		return fmt.Errorf("encode settings: %w", err)
	}

	const query = `
		INSERT INTO settings (key, value)
		VALUES ('app', ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value
	`

	if _, err := s.db.Exec(query, string(raw)); err != nil {
		return fmt.Errorf("save settings: %w", err)
	}

	return nil
}

func (s *SQLiteStore) GetReaderProgress(mangaID string) (contracts.ReaderProgress, bool, error) {
	const query = `
		SELECT manga_id, chapter_id, page, updated_at
		FROM reader_progress
		WHERE manga_id = ?
	`

	var progress contracts.ReaderProgress
	if err := s.db.QueryRow(query, mangaID).Scan(&progress.MangaID, &progress.ChapterID, &progress.Page, &progress.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return contracts.ReaderProgress{}, false, nil
		}

		return contracts.ReaderProgress{}, false, fmt.Errorf("select reader progress: %w", err)
	}

	return progress, true, nil
}

func (s *SQLiteStore) SaveReaderProgress(progress contracts.ReaderProgress) error {
	const query = `
		INSERT INTO reader_progress (manga_id, chapter_id, page, updated_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(manga_id) DO UPDATE SET
			chapter_id = excluded.chapter_id,
			page = excluded.page,
			updated_at = excluded.updated_at
	`

	if _, err := s.db.Exec(query, progress.MangaID, progress.ChapterID, progress.Page, progress.UpdatedAt); err != nil {
		return fmt.Errorf("save reader progress: %w", err)
	}

	return nil
}

func (s *SQLiteStore) SaveSiteProfile(profile contracts.SiteProfile) error {
	raw, err := json.Marshal(profile)
	if err != nil {
		return fmt.Errorf("encode site profile: %w", err)
	}

	const query = `
		INSERT INTO site_profiles (
			site,
			bundle_hash,
			bundle_url,
			profile_json,
			extracted_at
		)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(bundle_hash) DO UPDATE SET
			site = excluded.site,
			bundle_url = excluded.bundle_url,
			profile_json = excluded.profile_json,
			extracted_at = excluded.extracted_at
	`

	if _, err := s.db.Exec(
		query,
		profile.Site,
		profile.BundleHash,
		profile.BundleURL,
		string(raw),
		profile.ExtractedAt,
	); err != nil {
		return fmt.Errorf("save site profile: %w", err)
	}

	return nil
}

func (s *SQLiteStore) GetSiteProfile(bundleHash string) (contracts.SiteProfile, bool, error) {
	const query = `SELECT profile_json FROM site_profiles WHERE bundle_hash = ?`

	var raw string
	if err := s.db.QueryRow(query, bundleHash).Scan(&raw); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return contracts.SiteProfile{}, false, nil
		}

		return contracts.SiteProfile{}, false, fmt.Errorf("select site profile: %w", err)
	}

	var profile contracts.SiteProfile
	if err := json.Unmarshal([]byte(raw), &profile); err != nil {
		return contracts.SiteProfile{}, false, fmt.Errorf("decode site profile: %w", err)
	}

	return profile, true, nil
}

func (s *SQLiteStore) SaveDownloadJob(job contracts.DownloadJob, payloadJSON string) error {
	if payloadJSON == "" {
		payloadJSON = "{}"
	}

	const query = `
		INSERT INTO download_jobs (
			job_id,
			manga_slug,
			manga_title,
			source_url,
			status,
			queued_chapters,
			completed_chapters,
			failed_chapters,
			created_at,
			updated_at,
			last_error,
			max_concurrent_pages,
			payload_json
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(job_id) DO UPDATE SET
			manga_slug = excluded.manga_slug,
			manga_title = excluded.manga_title,
			source_url = excluded.source_url,
			status = excluded.status,
			queued_chapters = excluded.queued_chapters,
			completed_chapters = excluded.completed_chapters,
			failed_chapters = excluded.failed_chapters,
			updated_at = excluded.updated_at,
			last_error = excluded.last_error,
			max_concurrent_pages = excluded.max_concurrent_pages,
			payload_json = excluded.payload_json
	`

	if _, err := s.db.Exec(
		query,
		job.JobID,
		job.MangaSlug,
		job.MangaTitle,
		job.SourceURL,
		job.Status,
		job.QueuedChapters,
		job.CompletedChapters,
		job.FailedChapters,
		job.CreatedAt,
		job.UpdatedAt,
		job.LastError,
		job.MaxConcurrentPages,
		payloadJSON,
	); err != nil {
		return fmt.Errorf("save download job: %w", err)
	}

	return nil
}

func (s *SQLiteStore) GetDownloadJobPayload(jobID string) (string, bool, error) {
	const query = `SELECT payload_json FROM download_jobs WHERE job_id = ?`

	var payload string
	if err := s.db.QueryRow(query, jobID).Scan(&payload); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", false, nil
		}

		return "", false, fmt.Errorf("select download job payload: %w", err)
	}

	return payload, true, nil
}

func (s *SQLiteStore) ListDownloadJobs() ([]contracts.DownloadJob, error) {
	const query = `
		SELECT
			job_id,
			manga_slug,
			manga_title,
			source_url,
			status,
			queued_chapters,
			completed_chapters,
			failed_chapters,
			created_at,
			updated_at,
			last_error,
			max_concurrent_pages
		FROM download_jobs
		ORDER BY datetime(created_at) DESC
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("list download jobs: %w", err)
	}
	defer rows.Close()

	var jobs []contracts.DownloadJob
	for rows.Next() {
		var job contracts.DownloadJob
		if err := rows.Scan(
			&job.JobID,
			&job.MangaSlug,
			&job.MangaTitle,
			&job.SourceURL,
			&job.Status,
			&job.QueuedChapters,
			&job.CompletedChapters,
			&job.FailedChapters,
			&job.CreatedAt,
			&job.UpdatedAt,
			&job.LastError,
			&job.MaxConcurrentPages,
		); err != nil {
			return nil, fmt.Errorf("scan download job: %w", err)
		}

		jobs = append(jobs, job)
	}

	return jobs, rows.Err()
}

func (s *SQLiteStore) SaveLocalChapterState(mangaSlug string, chapterID string, sourceURL string, state contracts.LocalChapterState) error {
	raw, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("encode chapter state: %w", err)
	}

	const query = `
		INSERT INTO chapter_states (
			manga_slug,
			chapter_id,
			source_url,
			payload_json,
			updated_at
		)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(manga_slug, chapter_id) DO UPDATE SET
			source_url = excluded.source_url,
			payload_json = excluded.payload_json,
			updated_at = excluded.updated_at
	`

	if _, err := s.db.Exec(
		query,
		mangaSlug,
		chapterID,
		sourceURL,
		string(raw),
		time.Now().UTC().Format(time.RFC3339),
	); err != nil {
		return fmt.Errorf("save chapter state: %w", err)
	}

	return nil
}

func (s *SQLiteStore) ListLocalChapterStates(mangaSlug string) ([]contracts.LocalChapterState, error) {
	const query = `
		SELECT payload_json
		FROM chapter_states
		WHERE manga_slug = ?
		ORDER BY updated_at DESC
	`

	rows, err := s.db.Query(query, mangaSlug)
	if err != nil {
		return nil, fmt.Errorf("list chapter states: %w", err)
	}
	defer rows.Close()

	var states []contracts.LocalChapterState
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return nil, fmt.Errorf("scan chapter state: %w", err)
		}

		var state contracts.LocalChapterState
		if err := json.Unmarshal([]byte(raw), &state); err != nil {
			return nil, fmt.Errorf("decode chapter state: %w", err)
		}

		states = append(states, state)
	}

	return states, rows.Err()
}

func (s *SQLiteStore) SaveDownloadedFile(mangaSlug string, chapterID string, pageIndex int, filePath string, pageURL string, status string, sizeBytes int64) error {
	const query = `
		INSERT INTO downloaded_files (
			manga_slug,
			chapter_id,
			page_index,
			file_path,
			page_url,
			status,
			size_bytes,
			updated_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(manga_slug, chapter_id, page_index) DO UPDATE SET
			file_path = excluded.file_path,
			page_url = excluded.page_url,
			status = excluded.status,
			size_bytes = excluded.size_bytes,
			updated_at = excluded.updated_at
	`

	if _, err := s.db.Exec(
		query,
		mangaSlug,
		chapterID,
		pageIndex,
		filePath,
		pageURL,
		status,
		sizeBytes,
		time.Now().UTC().Format(time.RFC3339),
	); err != nil {
		return fmt.Errorf("save downloaded file: %w", err)
	}

	return nil
}
