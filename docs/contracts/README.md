# KLZ9 Downloader Contracts

This document is the single source of truth for the Wails method contracts, event
payloads, DTO names, enum values, and default settings used by the backend and
frontend implementations.

## Wails Methods

### `ResolveManga(inputURL string) -> ParsedMangaResult`

- Input: public `klz9.com` manga detail URL.
- Output: parsed manga metadata, chapter list, local state summary, and the
  active site profile reference.
- Errors:
  - `INVALID_URL`: URL is malformed or the slug cannot be inferred.
  - `PROFILE_NOT_FOUND`: bundle profile extraction failed or required fields are
    missing.
  - `MANGA_NOT_FOUND`: the site API returned `404`.
  - `BOOTSTRAP_FAILURE`: backend services were not initialized.

### `QueueChapters(req QueueDownloadRequest) -> DownloadJob`

- Input: a source URL, the resolved manga slug/title, the selected chapter IDs,
  and an output root override.
- Output: the newly created download job.
- Errors:
  - `CHAPTER_NOT_FOUND`: none of the requested chapter IDs matched.
  - `DOWNLOAD_FAILED`: job cannot be queued due to download initialization
    failure.
  - `BOOTSTRAP_FAILURE`: backend services were not initialized.

### `ListDownloadJobs() -> []DownloadJob`

- Output: all persisted download jobs ordered by creation time descending.
- Errors:
  - `STORE_FAILURE`: persisted job state could not be loaded.
  - `BOOTSTRAP_FAILURE`: backend services were not initialized.

### `PauseJob(jobID string) -> void`

- Effect: pause a running job. Running chapters will stop before downloading the
  next page.
- Errors:
  - `JOB_NOT_FOUND`: the requested job ID is unknown.
  - `BOOTSTRAP_FAILURE`: backend services were not initialized.

### `ResumeJob(jobID string) -> void`

- Effect: resume a paused job.
- Errors:
  - `JOB_NOT_FOUND`: the requested job ID is unknown.
  - `BOOTSTRAP_FAILURE`: backend services were not initialized.

### `RetryFailed(jobID string) -> void`

- Effect: reset failed chapters/pages for a job and schedule them again.
- Errors:
  - `JOB_NOT_FOUND`: the requested job ID is unknown.
  - `DOWNLOAD_FAILED`: the job is still running and cannot be retried yet.
  - `BOOTSTRAP_FAILURE`: backend services were not initialized.

### `ScanLocalState(sourceURL string) -> []LocalChapterState`

- Input: the manga source URL.
- Output: chapter-level local reconciliation results derived from SQLite and
  sidecar files.
- Errors:
  - `INVALID_URL`: URL is malformed or the slug cannot be inferred.
  - `MANGA_NOT_FOUND`: the site API returned `404`.
  - `BOOTSTRAP_FAILURE`: backend services were not initialized.

### `GetSettings() -> AppSettings`

- Output: the current persisted application settings.
- Errors:
  - `STORE_FAILURE`: settings could not be loaded.
  - `BOOTSTRAP_FAILURE`: backend services were not initialized.

### `UpdateSettings(settings AppSettings) -> AppSettings`

- Input: the complete settings object.
- Output: the normalized persisted settings.
- Errors:
  - `SETTINGS_INVALID`: unsupported enum value or invalid numeric setting.
  - `STORE_FAILURE`: settings could not be persisted.
  - `BOOTSTRAP_FAILURE`: backend services were not initialized.

### `GetReaderManifest(mangaID string) -> ReaderManifest`

- Input: local manga identifier from the library grid.
- Output: the chapter/page manifest used by the integrated reader.
- Errors:
  - `STORE_FAILURE`: local files or metadata could not be resolved.
  - `BOOTSTRAP_FAILURE`: backend services were not initialized.

### `GetReaderProgress(mangaID string) -> ReaderProgress`

- Input: local manga identifier from the library grid.
- Output: the last saved reader page for this manga. When no progress exists,
  `page=0`.
- Errors:
  - `STORE_FAILURE`: persisted progress could not be loaded.
  - `BOOTSTRAP_FAILURE`: backend services were not initialized.

### `UpdateReaderProgress(progress ReaderProgress) -> ReaderProgress`

- Input: the reader progress payload for one manga. Progress is page-precise,
  using a 1-based global page number.
- Output: the normalized persisted progress with the backend timestamp applied.
- Errors:
  - `STORE_FAILURE`: persisted progress could not be written.
  - `BOOTSTRAP_FAILURE`: backend services were not initialized.

## Events

### `download:job`

- Payload: `DownloadJob`
- Emitted when a job is created or its aggregate counters/status change.

### `download:chapter`

- Payload: `DownloadProgressEvent`
- Emitted when a chapter starts, pauses, resumes, completes, or fails.

### `download:page`

- Payload: `DownloadProgressEvent`
- Emitted for page-level progress updates during download.

### `library:reconciled`

- Payload: `ParsedMangaResult`
- Emitted after local files are reconciled against SQLite and sidecar data.

### `settings:updated`

- Payload: `AppSettings`
- Emitted after settings are persisted.

### `theme:resolved`

- Payload:
  - `mode`: `system | light | dark`
  - `resolved`: `light | dark`

### `locale:resolved`

- Payload:
  - `mode`: `system | manual`
  - `locale`: `zh-CN | en | ja`

## DTO Notes

### `AppSettings`

- `outputRoot`: absolute download root directory.
- `maxConcurrentDownloads`: page download concurrency used for future jobs.
- `retryCount`: per-page retry count. Actual attempts are `retryCount + 1`.
- `requestTimeoutSec`: HTTP timeout for HTML, API, and image requests.
- `localeMode`: `system | manual`.
- `locale`: `zh-CN | en | ja`.
- `themeMode`: `system | light | dark`.
- `readerScrollCachePages`: extra pages kept mounted before and after the
  viewport in virtual scroll mode.
- `autoRestoreReaderProgress`: whether the reader should reopen a manga at the
  last saved page.

### `ReaderProgress`

- `mangaID`: library manga identifier.
- `chapterID`: chapter containing the saved page.
- `page`: 1-based global page number across the full reader manifest.
- `updatedAt`: RFC3339 timestamp generated by the backend on save.

### `DownloadProgressEvent`

- `status`: chapter/page job status aligned to the shared `JobStatus` enum.
- `localStatus`: chapter-local filesystem status string.
- `completedAt`: RFC3339 timestamp for completed chapters only.

## Error Codes

- `INVALID_URL`: input URL is malformed or does not match the expected KLZ9
  manga/chapter shape. Frontend should show a direct inline validation error.
- `PROFILE_NOT_FOUND`: bundle protocol extraction failed. Frontend should show a
  retryable parse error and suggest reporting the changed site bundle.
- `MANGA_NOT_FOUND`: the API resource was not found. Frontend should show a
  terminal not-found state for the URL.
- `CHAPTER_NOT_FOUND`: selected chapters do not exist in the resolved result.
  Frontend should keep the page state and ask the user to reselect chapters.
- `DOWNLOAD_FAILED`: download or retry execution failed. Frontend should show
  the error on the affected job/chapter and keep retry actions enabled.
- `JOB_NOT_FOUND`: the requested job no longer exists. Frontend should refresh
  the queue list and clear stale local UI state.
- `STORE_FAILURE`: SQLite or sidecar persistence failed. Frontend should show a
  blocking storage error.
- `SETTINGS_INVALID`: one or more settings values are not supported. Frontend
  should highlight the form and keep the current saved settings.
- `BOOTSTRAP_FAILURE`: the desktop backend did not initialize. Frontend should
  show a blocking startup error and disable live actions.

## Enums

- `localeMode = system | manual`
- `locale = zh-CN | en | ja`
- `themeMode = system | light | dark`
- `local chapter status = not_downloaded | partial | complete | missing`
- `job status = queued | running | paused | completed | failed | canceled | skipped`

## Defaults

- `outputRoot = ~/Downloads/KLZ9`
- `maxConcurrentDownloads = 6`
- `retryCount = 3`
- `requestTimeoutSec = 30`
- `localeMode = system`
- `locale = en` as the persisted fallback; resolved locale still follows the
  host system when `localeMode=system`
- `themeMode = system`
- `readerScrollCachePages = 6`
- `autoRestoreReaderProgress = true`

## Protocol Inputs

- `index-*.js` participates in profile extraction.
- `vendor-*.js` does not participate in download protocol extraction.
- `*.css` files do not participate in download protocol extraction.
