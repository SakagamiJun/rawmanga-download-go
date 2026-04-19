export type LocaleMode = "system" | "manual";
export type Locale = "zh-CN" | "en" | "ja";
export type ThemeMode = "system" | "light" | "dark";
export type LocalChapterStatus = "not_downloaded" | "partial" | "complete" | "missing";
export type JobStatus = "queued" | "running" | "paused" | "completed" | "failed" | "canceled" | "skipped";

export interface SiteProfile {
  site: string;
  bundleURL: string;
  bundleHash: string;
  apiBase: string;
  signatureSecret: string;
  signatureMode: string;
  imageHostRewrite: Record<string, string>;
  ignorePageURLs: string[];
  extractedAt: string;
}

export interface ChapterItem {
  id: string;
  number: number;
  title: string;
  releaseDate: string;
  pageCount: number;
  localStatus: LocalChapterStatus;
  localPath: string;
  selected: boolean;
}

export interface LocalSummary {
  notDownloaded: number;
  partial: number;
  complete: number;
  missing: number;
}

export interface ParsedMangaResult {
  sourceURL: string;
  slug: string;
  title: string;
  coverURL: string;
  chapters: ChapterItem[];
  localSummary: LocalSummary;
  profileCacheHit: boolean;
  algorithmProfile: string;
}

export interface QueueDownloadRequest {
  sourceURL: string;
  mangaSlug: string;
  title: string;
  chapterIDs: string[];
  outputRoot: string;
}

export interface DownloadJob {
  jobID: string;
  mangaSlug: string;
  mangaTitle: string;
  sourceURL: string;
  status: JobStatus;
  queuedChapters: number;
  completedChapters: number;
  failedChapters: number;
  createdAt: string;
  updatedAt: string;
  lastError: string;
  maxConcurrentPages: number;
}

export interface DownloadProgressEvent {
  jobID: string;
  chapterID: string;
  chapterTitle: string;
  pageIndex: number;
  status: JobStatus;
  downloadedPages: number;
  totalPages: number;
  currentFile: string;
  bytesPerSec: number;
  error: string;
  localStatus: string;
  completedAt: string;
}

export interface LocalChapterState {
  chapterID: string;
  chapterNumber: number;
  title: string;
  status: LocalChapterStatus;
  localPath: string;
  localPageCount: number;
  expectedPageCount: number;
}

export interface AppSettings {
  outputRoot: string;
  maxConcurrentDownloads: number;
  retryCount: number;
  requestTimeoutSec: number;
  localeMode: LocaleMode;
  locale: Locale;
  themeMode: ThemeMode;
}

export interface LibraryManga {
  id: string;
  title: string;
  relativePath: string;
  coverImageURL: string;
  chapterCount: number;
  pageCount: number;
  lastUpdated: string;
}

export interface ReaderManifest {
  mangaID: string;
  title: string;
  coverImageURL: string;
  totalPages: number;
  chapters: ReaderChapter[];
}

export interface ReaderChapter {
  id: string;
  title: string;
  number: number;
  startPage: number;
  pageCount: number;
  pages: ReaderPage[];
  localPath: string;
  completedAt: string;
}

export interface ReaderPage {
  id: string;
  chapterID: string;
  chapterTitle: string;
  pageIndex: number;
  fileName: string;
  sourceURL: string;
}

export const EVENTS = {
  DOWNLOAD_JOB: "download:job",
  DOWNLOAD_CHAPTER: "download:chapter",
  DOWNLOAD_PAGE: "download:page",
  LIBRARY_RECONCILED: "library:reconciled",
  SETTINGS_UPDATED: "settings:updated",
  THEME_RESOLVED: "theme:resolved",
  LOCALE_RESOLVED: "locale:resolved",
} as const;
