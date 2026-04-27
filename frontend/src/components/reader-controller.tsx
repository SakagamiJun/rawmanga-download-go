import { useEffect, useMemo, useRef, useState } from "react";
import { PagedReader } from "@/components/paged-reader";
import { ScrollReader } from "@/components/scroll-reader";
import { type FlatReaderPage, type PageMetric, type ReaderNavigationRequest, DEFAULT_ASPECT_RATIO, clampIndex } from "@/components/reader-shared";
import { appAdapter } from "@/lib/api";
import type { AppSettings, ReaderManifest } from "@/lib/contracts";

export type ReaderMode = "scroll" | "paged";

export type ReaderJumpRequest =
  | {
      requestID: number;
      target: "chapter";
      chapterID: string;
    }
  | {
      requestID: number;
      target: "page";
      page: number;
    };

interface ReaderControllerProps {
  jumpRequest?: ReaderJumpRequest | null;
  manifest: ReaderManifest;
  mode: ReaderMode;
  settings: Pick<AppSettings, "autoRestoreReaderProgress" | "readerScrollCachePages">;
}

function resolveJumpTargetIndex(jumpRequest: ReaderJumpRequest, pages: FlatReaderPage[]) {
  if (jumpRequest.target === "chapter") {
    return pages.findIndex((page) => page.chapterID === jumpRequest.chapterID);
  }

  return clampIndex(jumpRequest.page - 1, pages.length);
}

export function ReaderController({ jumpRequest = null, manifest, mode, settings }: ReaderControllerProps) {
  const handledJumpRequestIDRef = useRef<number | null>(null);
  const lastSavedPageRef = useRef<number | null>(null);
  const metricRequestGenerationRef = useRef(0);
  const navigationRequestIDRef = useRef(0);
  const previousModeRef = useRef(mode);
  const requestedMetricIDsRef = useRef<Set<string>>(new Set());

  const [metrics, setMetrics] = useState<Record<string, PageMetric>>({});
  const [currentIndex, setCurrentIndex] = useState(0);
  const [navigationRequest, setNavigationRequest] = useState<ReaderNavigationRequest | null>(null);
  const [restoreReady, setRestoreReady] = useState(false);

  const pages = useMemo<FlatReaderPage[]>(() => {
    let globalIndex = 0;
    return manifest.chapters.flatMap((chapter) =>
      chapter.pages.map((page) => {
        const item: FlatReaderPage = {
          ...page,
          globalIndex,
          globalPage: globalIndex + 1,
        };
        globalIndex += 1;
        return item;
      })
    );
  }, [manifest]);

  const cacheRadius = Math.max(1, settings.readerScrollCachePages || 6);

  const nextNavigationRequest = (index: number, reason: ReaderNavigationRequest["reason"]) => {
    navigationRequestIDRef.current += 1;
    return {
      id: navigationRequestIDRef.current,
      index: clampIndex(index, pages.length),
      reason,
    } satisfies ReaderNavigationRequest;
  };

  const onMetricMeasured = (pageID: string, width: number, height: number) => {
    if (!width || !height) {
      return;
    }

    setMetrics((current) => {
      const previousMetric = current[pageID];
      if (previousMetric && previousMetric.width === width && previousMetric.height === height) {
        return current;
      }

      return {
        ...current,
        [pageID]: { width, height },
      };
    });
  };

  const requestMetric = (page: FlatReaderPage | undefined) => {
    if (!page || metrics[page.id] || requestedMetricIDsRef.current.has(page.id)) {
      return;
    }

    const requestGeneration = metricRequestGenerationRef.current;
    requestedMetricIDsRef.current.add(page.id);

    const image = new Image();
    image.src = page.sourceURL;
    image.onload = () => {
      if (metricRequestGenerationRef.current !== requestGeneration) {
        return;
      }

      onMetricMeasured(page.id, image.naturalWidth || DEFAULT_ASPECT_RATIO, image.naturalHeight || 1);
    };
    image.onerror = () => {
      if (metricRequestGenerationRef.current !== requestGeneration) {
        return;
      }

      onMetricMeasured(page.id, DEFAULT_ASPECT_RATIO, 1);
    };
  };

  useEffect(() => {
    metricRequestGenerationRef.current += 1;
    requestedMetricIDsRef.current = new Set();
    handledJumpRequestIDRef.current = null;
    lastSavedPageRef.current = null;
    previousModeRef.current = mode;

    setMetrics({});
    setCurrentIndex(0);
    setNavigationRequest(null);
    setRestoreReady(false);
  }, [manifest.mangaID]);

  useEffect(() => {
    if (pages.length === 0) {
      setRestoreReady(true);
      return;
    }

    if (!settings.autoRestoreReaderProgress) {
      setRestoreReady(true);
      return;
    }

    let cancelled = false;
    setRestoreReady(false);

    void appAdapter
      .getReaderProgress(manifest.mangaID)
      .then((progress) => {
        if (cancelled) {
          return;
        }

        if (progress.page > 0) {
          const targetIndex = clampIndex(progress.page - 1, pages.length);
          lastSavedPageRef.current = targetIndex + 1;
          setCurrentIndex(targetIndex);
          setNavigationRequest(nextNavigationRequest(targetIndex, "restore"));
        }

        setRestoreReady(true);
      })
      .catch(() => {
        if (!cancelled) {
          setRestoreReady(true);
        }
      });

    return () => {
      cancelled = true;
    };
  }, [manifest.mangaID, pages.length, settings.autoRestoreReaderProgress]);

  useEffect(() => {
    if (previousModeRef.current === mode || pages.length === 0) {
      previousModeRef.current = mode;
      return;
    }

    previousModeRef.current = mode;
    setNavigationRequest(nextNavigationRequest(currentIndex, "sync"));
  }, [currentIndex, mode, pages.length]);

  useEffect(() => {
    if (!jumpRequest || pages.length === 0 || handledJumpRequestIDRef.current === jumpRequest.requestID) {
      return;
    }

    handledJumpRequestIDRef.current = jumpRequest.requestID;

    const targetIndex = resolveJumpTargetIndex(jumpRequest, pages);

    if (targetIndex < 0) {
      return;
    }

    setCurrentIndex(targetIndex);
    setNavigationRequest(nextNavigationRequest(targetIndex, mode === "scroll" ? "sync" : "jump"));
  }, [jumpRequest, mode, pages]);

  useEffect(() => {
    const start = Math.max(0, currentIndex - cacheRadius);
    const end = Math.min(pages.length - 1, currentIndex + cacheRadius);

    for (let index = start; index <= end; index += 1) {
      requestMetric(pages[index]);
    }
  }, [cacheRadius, currentIndex, pages, requestMetric]);

  useEffect(() => {
    if (!restoreReady || pages.length === 0) {
      return;
    }

    const activePage = pages[clampIndex(currentIndex, pages.length)];
    if (!activePage || lastSavedPageRef.current === activePage.globalPage) {
      return;
    }

    const timer = window.setTimeout(() => {
      lastSavedPageRef.current = activePage.globalPage;
      void appAdapter.updateReaderProgress({
        mangaID: manifest.mangaID,
        chapterID: activePage.chapterID,
        page: activePage.globalPage,
        updatedAt: "",
      });
    }, 300);

    return () => window.clearTimeout(timer);
  }, [currentIndex, manifest.mangaID, pages, restoreReady]);

  const handleCurrentIndexChange = (nextIndex: number) => {
    const clampedIndex = clampIndex(nextIndex, pages.length);
    setCurrentIndex((current) => (current === clampedIndex ? current : clampedIndex));
  };

  if (pages.length === 0) {
    return (
      <div className="flex h-full items-center justify-center border-l border-border/40 bg-card/14 text-sm text-muted-foreground">
        No local pages available for this manga yet.
      </div>
    );
  }

  if (mode === "scroll") {
    return (
      <ScrollReader
        cacheRadius={cacheRadius}
        currentIndex={currentIndex}
        metrics={metrics}
        navigationRequest={navigationRequest}
        onCurrentIndexChange={handleCurrentIndexChange}
        onMetricMeasured={onMetricMeasured}
        pages={pages}
        requestMetric={requestMetric}
      />
    );
  }

  return (
    <PagedReader
      currentIndex={currentIndex}
      metrics={metrics}
      navigationRequest={navigationRequest}
      onCurrentIndexChange={handleCurrentIndexChange}
      onMetricMeasured={onMetricMeasured}
      pages={pages}
      requestMetric={requestMetric}
    />
  );
}
