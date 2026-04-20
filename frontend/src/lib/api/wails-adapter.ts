import type {
  AppSettings,
  DownloadJob,
  LibraryManga,
  LocalChapterState,
  ParsedMangaResult,
  QueueDownloadRequest,
  ReaderManifest,
  ReaderProgress,
} from "@/lib/contracts";
import type { AppAdapter } from "@/lib/api/adapter";
import { getWailsApp, getWailsRuntime } from "@/lib/runtime";

export class WailsAdapter implements AppAdapter {
  readonly mode = "wails" as const;

  async resolveManga(inputURL: string) {
    return (await getWailsApp()?.ResolveManga?.(inputURL)) as ParsedMangaResult;
  }

  async queueChapters(request: QueueDownloadRequest) {
    return (await getWailsApp()?.QueueChapters?.(request)) as DownloadJob;
  }

  async listDownloadJobs() {
    return ((await getWailsApp()?.ListDownloadJobs?.()) ?? []) as DownloadJob[];
  }

  async pauseJob(jobID: string) {
    await getWailsApp()?.PauseJob?.(jobID);
  }

  async resumeJob(jobID: string) {
    await getWailsApp()?.ResumeJob?.(jobID);
  }

  async retryFailed(jobID: string) {
    await getWailsApp()?.RetryFailed?.(jobID);
  }

  async scanLocalState(sourceURL: string) {
    return ((await getWailsApp()?.ScanLocalState?.(sourceURL)) ?? []) as LocalChapterState[];
  }

  async getSettings() {
    return (await getWailsApp()?.GetSettings?.()) as AppSettings;
  }

  async updateSettings(input: AppSettings) {
    return (await getWailsApp()?.UpdateSettings?.(input)) as AppSettings;
  }

  async listLibraryManga() {
    return ((await getWailsApp()?.ListLibraryManga?.()) ?? []) as LibraryManga[];
  }

  async getReaderManifest(mangaID: string) {
    return (await getWailsApp()?.GetReaderManifest?.(mangaID)) as ReaderManifest;
  }

  async getReaderProgress(mangaID: string) {
    return (await getWailsApp()?.GetReaderProgress?.(mangaID)) as ReaderProgress;
  }

  async updateReaderProgress(input: ReaderProgress) {
    return (await getWailsApp()?.UpdateReaderProgress?.(input)) as ReaderProgress;
  }

  subscribe(eventName: string, callback: (payload: unknown) => void) {
    const runtime = getWailsRuntime();
    const unsubscribe = runtime?.EventsOn?.(eventName, callback);
    if (typeof unsubscribe === "function") {
      return unsubscribe;
    }
    return () => runtime?.EventsOff?.(eventName);
  }
}
