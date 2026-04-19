# KLZ9 Downloader

Desktop downloader for `klz9.com` manga pages built with Go, Wails, React,
TypeScript, Tailwind, and lightweight shared contracts between backend and
frontend.

## Current V1 Scope

- Resolve a KLZ9 manga URL into chapter metadata.
- Extract the active protocol profile from `assets/index-*.js`.
- Cache extracted `SiteProfile` data in SQLite and on disk by bundle hash.
- Download selected chapters into manga/chapter folders with atomic file writes.
- Track local state in SQLite plus per-chapter `.klz9-chapter.json` sidecars.
- Avoid re-downloading complete files and classify chapters as
  `not_downloaded | partial | complete | missing`.
- Support configurable page concurrency, retry count, timeout, locale, and
  theme.
- Support `zh-CN`, `en`, `ja`, with default locale mode `system`.
- Support `light`, `dark`, `system`, with default theme mode `system`.

Out of scope for v1:

- Execution-sandbox caching.
- Login, cookie import, or VIP-only content.
- Protocol extraction from `vendor-*.js` or `*.css`.

## Architecture

- `internal/contracts`: shared Go DTOs, enums, errors, and event names.
- `internal/klz9`: KLZ9 protocol extraction, profile cache, signed API client.
- `internal/download`: queueing, concurrency, retries, atomic page writes, and
  sidecars.
- `internal/store`: SQLite persistence for settings, jobs, files, local state,
  and profile indices.
- `internal/settings`: settings defaults, normalization, and persistence.
- `frontend/src`: React UI, adapter layer, i18n, theme resolution, and local UI
  state.
- `docs/contracts`: frozen method/event contracts and sample payloads.

## Protocol Notes

- `index-*.js` is the only bundle used for protocol profile extraction.
- `vendor-CjW7dN_t.js` is treated as third-party runtime code and is not part of
  the downloader protocol implementation.
- `A.index-BTZoaLqr.css.pagespeed.cf.e2sct_UovP.css` is not used for download
  protocol extraction.
- Profile extraction caches `apiBase`, signature secret/mode, image host
  rewrites, and ignore-page lists by bundle hash.

## Defaults

- Max concurrent downloads: `6`
- Retry count: `3`
- Request timeout: `30` seconds
- Locale mode: `system`
- Theme mode: `system`

## Development

Prerequisites:

- Go `1.26`
- `pnpm`
- Wails CLI if you want live desktop development

Install dependencies:

```bash
go mod tidy
cd frontend
pnpm install
```

Run checks:

```bash
go test ./...
cd frontend
pnpm test
pnpm build
```

Run the app:

```bash
wails dev
```

If you only want to verify the embedded build path, build the frontend first and
then run:

```bash
pnpm -C frontend build
go run .
```

## Frontend Data Source

- In Wails runtime, the frontend uses the live adapter and calls the frozen
  methods on `App`.
- Outside Wails, the frontend falls back to a mock adapter so the UI can still
  be developed in isolation.

## Acceptance Checklist

- Resolve `https://klz9.com/otona-ni-narenai-bokura-wa.html` into chapters.
- Select some or all chapters for download.
- Download images into manga/chapter folders with stable file names.
- Avoid duplicate downloads for already complete files.
- Detect `partial` and `missing` states from local files plus sidecars.
- Resume by downloading only missing pages.
- Apply locale switching for `zh-CN`, `en`, `ja`.
- Apply theme switching for `light`, `dark`, `system`.
- Apply updated concurrency settings to future jobs.
