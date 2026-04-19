import type {
  AppSettings,
  DownloadJob,
  DownloadProgressEvent,
  LibraryManga,
  LocalChapterState,
  ParsedMangaResult,
  QueueDownloadRequest,
  ReaderManifest,
} from "@/lib/contracts";

export interface AppAdapter {
  readonly mode: "mock" | "wails";
  resolveManga(inputURL: string): Promise<ParsedMangaResult>;
  queueChapters(request: QueueDownloadRequest): Promise<DownloadJob>;
  listDownloadJobs(): Promise<DownloadJob[]>;
  pauseJob(jobID: string): Promise<void>;
  resumeJob(jobID: string): Promise<void>;
  retryFailed(jobID: string): Promise<void>;
  scanLocalState(sourceURL: string): Promise<LocalChapterState[]>;
  getSettings(): Promise<AppSettings>;
  updateSettings(input: AppSettings): Promise<AppSettings>;
  listLibraryManga(): Promise<LibraryManga[]>;
  getReaderManifest(mangaID: string): Promise<ReaderManifest>;
  subscribe(eventName: string, callback: (payload: unknown) => void): () => void;
}
