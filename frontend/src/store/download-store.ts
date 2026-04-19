import { create } from "zustand";
import type { DownloadJob, DownloadProgressEvent } from "@/lib/contracts";

function isQueueVisibleJob(job: DownloadJob) {
  return job.status !== "completed" && job.status !== "canceled" && job.status !== "skipped";
}

interface DownloadState {
  jobs: Record<string, DownloadJob>;
  progress: Record<string, DownloadProgressEvent>;
  seedJobs: (jobs: DownloadJob[]) => void;
  mergeJob: (job: DownloadJob) => void;
  mergeProgress: (event: DownloadProgressEvent) => void;
}

export const useDownloadStore = create<DownloadState>((set) => ({
  jobs: {},
  progress: {},
  seedJobs: (jobs) =>
    set(() => ({
      jobs: Object.fromEntries(jobs.filter(isQueueVisibleJob).map((job) => [job.jobID, job])),
    })),
  mergeJob: (job) =>
    set((state) => {
      if (!isQueueVisibleJob(job)) {
        const nextJobs = { ...state.jobs };
        delete nextJobs[job.jobID];

        const nextProgress = Object.fromEntries(
          Object.entries(state.progress).filter(([key]) => !key.startsWith(`${job.jobID}:`))
        );

        return {
          jobs: nextJobs,
          progress: nextProgress,
        };
      }

      return {
        jobs: {
          ...state.jobs,
          [job.jobID]: job,
        },
      };
    }),
  mergeProgress: (event) =>
    set((state) => ({
      progress: {
        ...state.progress,
        [`${event.jobID}:${event.chapterID}`]: event,
      },
    })),
}));
