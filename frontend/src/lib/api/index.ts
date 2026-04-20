import { getWailsApp } from "@/lib/runtime";
import type { AppAdapter } from "@/lib/api/adapter";
import { MockAdapter } from "@/lib/api/mock-adapter";
import { WailsAdapter } from "@/lib/api/wails-adapter";

const mockAdapter = new MockAdapter();
const wailsAdapter = new WailsAdapter();

function currentAdapter(): AppAdapter {
  return getWailsApp() ? wailsAdapter : mockAdapter;
}

export const appAdapter: AppAdapter = {
  get mode() {
    return currentAdapter().mode;
  },
  resolveManga(inputURL) {
    return currentAdapter().resolveManga(inputURL);
  },
  queueChapters(request) {
    return currentAdapter().queueChapters(request);
  },
  listDownloadJobs() {
    return currentAdapter().listDownloadJobs();
  },
  pauseJob(jobID) {
    return currentAdapter().pauseJob(jobID);
  },
  resumeJob(jobID) {
    return currentAdapter().resumeJob(jobID);
  },
  retryFailed(jobID) {
    return currentAdapter().retryFailed(jobID);
  },
  scanLocalState(sourceURL) {
    return currentAdapter().scanLocalState(sourceURL);
  },
  getSettings() {
    return currentAdapter().getSettings();
  },
  updateSettings(input) {
    return currentAdapter().updateSettings(input);
  },
  listLibraryManga() {
    return currentAdapter().listLibraryManga();
  },
  getReaderManifest(mangaID) {
    return currentAdapter().getReaderManifest(mangaID);
  },
  getReaderProgress(mangaID) {
    return currentAdapter().getReaderProgress(mangaID);
  },
  updateReaderProgress(input) {
    return currentAdapter().updateReaderProgress(input);
  },
  subscribe(eventName, callback) {
    return currentAdapter().subscribe(eventName, callback);
  },
};
