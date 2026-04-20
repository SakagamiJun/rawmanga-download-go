import {
  type AppSettings,
  type DownloadJob,
  type DownloadProgressEvent,
  EVENTS,
  type LibraryManga,
  type LocalChapterState,
  type ParsedMangaResult,
  type QueueDownloadRequest,
  type ReaderManifest,
  type ReaderProgress,
} from "@/lib/contracts";
import type { AppAdapter } from "@/lib/api/adapter";

const STORAGE_KEY = "klz9-downloader-settings";
const READER_PROGRESS_STORAGE_KEY = "klz9-reader-progress";

const sampleManga: ParsedMangaResult = {
  sourceURL: "https://klz9.com/otona-ni-narenai-bokura-wa.html",
  slug: "otona-ni-narenai-bokura-wa",
  title: "Otona ni Narenai Bokura wa",
  coverURL: "https://picsum.photos/seed/klz9-cover-1/1200/1800",
  chapters: Array.from({ length: 12 }, (_, index) => ({
    id: String(index + 14),
    number: 25 - index,
    title: `Chapter ${25 - index}`,
    releaseDate: new Date(Date.now() - index * 86400000).toISOString(),
    pageCount: 24 + (index % 5),
    localStatus: index < 2 ? "complete" : index === 2 ? "partial" : "not_downloaded",
    localPath: index < 3 ? `/Users/example/Downloads/KLZ9/Otona/0${25 - index} - Chapter ${25 - index}` : "",
    selected: false,
  })),
  localSummary: {
    notDownloaded: 9,
    partial: 1,
    complete: 2,
    missing: 0,
  },
  profileCacheHit: true,
  algorithmProfile: "klz9:index-BBvPdTHw.js.pagespeed.ce.WKBIIa11t7.js",
};

const defaultSettings: AppSettings = {
  outputRoot: "/Users/example/Downloads/KLZ9",
  maxConcurrentDownloads: 6,
  retryCount: 3,
  requestTimeoutSec: 30,
  localeMode: "system",
  locale: "en",
  themeMode: "system",
  readerScrollCachePages: 6,
  autoRestoreReaderProgress: true,
};

function createMockReaderManifest(index: number, title: string): ReaderManifest {
  const chapters = Array.from({ length: 4 }, (_, chapterIndex) => {
    const pages = Array.from({ length: 10 }, (_, pageIndex) => ({
      id: `m${index}-c${chapterIndex + 1}-p${pageIndex + 1}`,
      chapterID: `m${index}-chapter-${chapterIndex + 1}`,
      chapterTitle: `Chapter ${chapterIndex + 1}`,
      pageIndex,
      fileName: `${String(pageIndex + 1).padStart(3, "0")}.jpg`,
      sourceURL: `https://picsum.photos/seed/klz9-${index}-${chapterIndex + 1}-${pageIndex + 1}/1400/2000`,
    }));

    return {
      id: `m${index}-chapter-${chapterIndex + 1}`,
      title: `Chapter ${chapterIndex + 1}`,
      number: chapterIndex + 1,
      startPage: chapterIndex * 10,
      pageCount: pages.length,
      pages,
      localPath: `/Users/example/Downloads/KLZ9/${title}/00${chapterIndex + 1} - Chapter ${chapterIndex + 1}`,
      completedAt: new Date(Date.now() - chapterIndex * 86400000).toISOString(),
    };
  });

  return {
    mangaID: `mock-library-${index}`,
    title,
    coverImageURL: chapters[0]?.pages[0]?.sourceURL ?? "",
    totalPages: chapters.reduce((sum, chapter) => sum + chapter.pages.length, 0),
    chapters,
  };
}

const mockReaderManifests: ReaderManifest[] = [
  createMockReaderManifest(1, "Otona ni Narenai Bokura wa"),
  createMockReaderManifest(2, "Midnight Signal"),
  createMockReaderManifest(3, "Glass Archive"),
];

const mockLibrary: LibraryManga[] = mockReaderManifests.map((manifest, index) => ({
  id: manifest.mangaID,
  title: manifest.title,
  sourceURL: `https://klz9.com/mock-library-${index + 1}.html`,
  relativePath: manifest.title,
  coverImageURL: manifest.coverImageURL,
  chapterCount: manifest.chapters.length,
  pageCount: manifest.totalPages,
  lastUpdated: new Date(Date.now() - index * 172800000).toISOString(),
}));

type Listener = (payload: unknown) => void;

export class MockAdapter implements AppAdapter {
  readonly mode = "mock" as const;

  private listeners = new Map<string, Set<Listener>>();
  private jobs = new Map<string, DownloadJob>();
  private timers = new Map<string, number>();
  private pauseFlags = new Set<string>();

  async resolveManga(inputURL: string) {
    return {
      ...sampleManga,
      sourceURL: inputURL,
    };
  }

  async queueChapters(request: QueueDownloadRequest) {
    const job: DownloadJob = {
      jobID: `mock-${crypto.randomUUID()}`,
      mangaSlug: request.mangaSlug,
      mangaTitle: request.title,
      sourceURL: request.sourceURL,
      status: "queued",
      queuedChapters: request.chapterIDs.length,
      completedChapters: 0,
      failedChapters: 0,
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
      lastError: "",
      maxConcurrentPages: this.readSettings().maxConcurrentDownloads,
    };

    this.jobs.set(job.jobID, job);
    this.emit(EVENTS.DOWNLOAD_JOB, job);
    this.simulateJob(job, request);
    return job;
  }

  async listDownloadJobs() {
    return Array.from(this.jobs.values()).sort((left, right) => right.createdAt.localeCompare(left.createdAt));
  }

  async pauseJob(jobID: string) {
    this.pauseFlags.add(jobID);
    const job = this.jobs.get(jobID);
    if (!job) {
      return;
    }
    const updated = { ...job, status: "paused" as const, updatedAt: new Date().toISOString() };
    this.jobs.set(jobID, updated);
    this.emit(EVENTS.DOWNLOAD_JOB, updated);
  }

  async resumeJob(jobID: string) {
    this.pauseFlags.delete(jobID);
    const job = this.jobs.get(jobID);
    if (!job) {
      return;
    }
    const updated = { ...job, status: "running" as const, updatedAt: new Date().toISOString() };
    this.jobs.set(jobID, updated);
    this.emit(EVENTS.DOWNLOAD_JOB, updated);
  }

  async retryFailed(jobID: string) {
    const job = this.jobs.get(jobID);
    if (!job) {
      return;
    }
    const updated = {
      ...job,
      status: "queued" as const,
      failedChapters: 0,
      completedChapters: 0,
      lastError: "",
      updatedAt: new Date().toISOString(),
    };
    this.jobs.set(jobID, updated);
    this.emit(EVENTS.DOWNLOAD_JOB, updated);
  }

  async scanLocalState() {
    return sampleManga.chapters.map<LocalChapterState>((chapter) => ({
      chapterID: chapter.id,
      chapterNumber: chapter.number,
      title: chapter.title,
      status: chapter.localStatus,
      localPath: chapter.localPath,
      localPageCount: chapter.localStatus === "complete" ? chapter.pageCount : Math.max(0, chapter.pageCount - 5),
      expectedPageCount: chapter.pageCount,
    }));
  }

  async getSettings() {
    return this.readSettings();
  }

  async updateSettings(input: AppSettings) {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(input));
    this.emit(EVENTS.SETTINGS_UPDATED, input);
    return input;
  }

  async listLibraryManga() {
    return mockLibrary;
  }

  async getReaderManifest(mangaID: string) {
    return mockReaderManifests.find((manifest) => manifest.mangaID === mangaID) ?? mockReaderManifests[0];
  }

  async getReaderProgress(mangaID: string) {
    return this.readReaderProgress(mangaID);
  }

  async updateReaderProgress(input: ReaderProgress) {
    const progress: ReaderProgress = {
      mangaID: input.mangaID,
      chapterID: input.chapterID,
      page: Math.max(1, input.page),
      updatedAt: new Date().toISOString(),
    };

    const allProgress = this.readAllReaderProgress();
    allProgress[progress.mangaID] = progress;
    localStorage.setItem(READER_PROGRESS_STORAGE_KEY, JSON.stringify(allProgress));

    return progress;
  }

  subscribe(eventName: string, callback: Listener) {
    const listeners = this.listeners.get(eventName) ?? new Set<Listener>();
    listeners.add(callback);
    this.listeners.set(eventName, listeners);
    return () => {
      listeners.delete(callback);
    };
  }

  private readSettings(): AppSettings {
    try {
      const raw = localStorage.getItem(STORAGE_KEY);
      if (!raw) {
        return defaultSettings;
      }
      return { ...defaultSettings, ...(JSON.parse(raw) as AppSettings) };
    } catch {
      return defaultSettings;
    }
  }

  private readReaderProgress(mangaID: string): ReaderProgress {
    return (
      this.readAllReaderProgress()[mangaID] ?? {
        mangaID,
        chapterID: "",
        page: 0,
        updatedAt: "",
      }
    );
  }

  private readAllReaderProgress(): Record<string, ReaderProgress> {
    try {
      const raw = localStorage.getItem(READER_PROGRESS_STORAGE_KEY);
      if (!raw) {
        return {};
      }

      return JSON.parse(raw) as Record<string, ReaderProgress>;
    } catch {
      return {};
    }
  }

  private emit(eventName: string, payload: unknown) {
    this.listeners.get(eventName)?.forEach((listener) => listener(payload));
  }

  private simulateJob(job: DownloadJob, request: QueueDownloadRequest) {
    const chapters = sampleManga.chapters.filter((chapter) => request.chapterIDs.includes(chapter.id));
    let chapterIndex = 0;

    const tick = () => {
      const currentJob = this.jobs.get(job.jobID);
      if (!currentJob) {
        return;
      }
      if (this.pauseFlags.has(job.jobID)) {
        this.timers.set(job.jobID, window.setTimeout(tick, 300));
        return;
      }

      if (chapterIndex >= chapters.length) {
        const completedJob = {
          ...currentJob,
          status: "completed" as const,
          completedChapters: chapters.length,
          updatedAt: new Date().toISOString(),
        };
        this.jobs.set(job.jobID, completedJob);
        this.emit(EVENTS.DOWNLOAD_JOB, completedJob);
        return;
      }

      const activeChapter = chapters[chapterIndex];
      const runningJob = { ...currentJob, status: "running" as const, updatedAt: new Date().toISOString() };
      this.jobs.set(job.jobID, runningJob);
      this.emit(EVENTS.DOWNLOAD_JOB, runningJob);

      let downloadedPages = 0;
      const pageTick = () => {
        if (this.pauseFlags.has(job.jobID)) {
          this.timers.set(job.jobID, window.setTimeout(pageTick, 300));
          return;
        }

        downloadedPages += 1;
        const event: DownloadProgressEvent = {
          jobID: job.jobID,
          chapterID: activeChapter.id,
          chapterTitle: activeChapter.title,
          pageIndex: downloadedPages - 1,
          status: "running",
          downloadedPages,
          totalPages: activeChapter.pageCount,
          currentFile: `${String(downloadedPages).padStart(3, "0")}.jpg`,
          bytesPerSec: 512000,
          error: "",
          localStatus: downloadedPages >= activeChapter.pageCount ? "complete" : "partial",
          completedAt: downloadedPages >= activeChapter.pageCount ? new Date().toISOString() : "",
        };
        this.emit(EVENTS.DOWNLOAD_PAGE, event);
        this.emit(EVENTS.DOWNLOAD_CHAPTER, event);

        if (downloadedPages >= activeChapter.pageCount) {
          const nextJob = {
            ...runningJob,
            completedChapters: chapterIndex + 1,
            updatedAt: new Date().toISOString(),
          };
          this.jobs.set(job.jobID, nextJob);
          this.emit(EVENTS.DOWNLOAD_JOB, nextJob);
          chapterIndex += 1;
          this.timers.set(job.jobID, window.setTimeout(tick, 250));
          return;
        }

        this.timers.set(job.jobID, window.setTimeout(pageTick, 120));
      };

      pageTick();
    };

    this.timers.set(job.jobID, window.setTimeout(tick, 200));
  }
}
