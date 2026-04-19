import { type KeyboardEvent, type WheelEvent, useEffect, useMemo, useRef, useState } from "react";
import type { ReaderManifest, ReaderPage } from "@/lib/contracts";

type ReaderMode = "scroll" | "paged";

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

interface ReaderViewProps {
  manifest: ReaderManifest;
  jumpRequest?: ReaderJumpRequest | null;
  mode: ReaderMode;
}

interface PageMetric {
  width: number;
  height: number;
}

interface FlatReaderPage extends ReaderPage {
  globalIndex: number;
}

const DEFAULT_ASPECT_RATIO = 0.72;
const PAGE_GAP = 0;
const PAGE_PADDING = 0;

export function ReaderView({ manifest, jumpRequest = null, mode }: ReaderViewProps) {
  const containerRef = useRef<HTMLDivElement | null>(null);
  const scrollRef = useRef<HTMLDivElement | null>(null);
  const handledJumpRequestIDRef = useRef<number | null>(null);
  const pendingJumpIndexRef = useRef<number | null>(null);
  const metricRequestGenerationRef = useRef(0);
  const requestedMetricIDsRef = useRef<Set<string>>(new Set());

  const [viewportWidth, setViewportWidth] = useState(0);
  const [viewportHeight, setViewportHeight] = useState(0);
  const [metrics, setMetrics] = useState<Record<string, PageMetric>>({});
  const [currentIndex, setCurrentIndex] = useState(0);
  const [scrollTop, setScrollTop] = useState(0);

  const pages = useMemo<FlatReaderPage[]>(() => {
    let globalIndex = 0;
    return manifest.chapters.flatMap((chapter) =>
      chapter.pages.map((page) => {
        const item: FlatReaderPage = {
          ...page,
          globalIndex,
        };
        globalIndex += 1;
        return item;
      })
    );
  }, [manifest]);

  useEffect(() => {
    metricRequestGenerationRef.current += 1;
    requestedMetricIDsRef.current = new Set();
    pendingJumpIndexRef.current = null;
    setMetrics({});
    setCurrentIndex(0);
    setScrollTop(0);
    if (scrollRef.current) {
      scrollRef.current.scrollTop = 0;
    }
  }, [manifest.mangaID, mode]);

  useEffect(() => {
    const element = containerRef.current;
    if (!element) {
      return;
    }

    const syncSize = () => {
      setViewportWidth(element.clientWidth);
      setViewportHeight(element.clientHeight);
    };

    syncSize();
    const observer = new ResizeObserver(syncSize);
    observer.observe(element);
    return () => observer.disconnect();
  }, []);

  const contentWidth = Math.max(320, viewportWidth - PAGE_PADDING * 2);
  const contentHeight = Math.max(320, viewportHeight - PAGE_PADDING * 2);

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
      setMetrics((current) => {
        if (current[page.id]) {
          return current;
        }
        return {
          ...current,
          [page.id]: {
            width: image.naturalWidth || 1,
            height: image.naturalHeight || 1,
          },
        };
      });
    };
    image.onerror = () => {
      if (metricRequestGenerationRef.current !== requestGeneration) {
        return;
      }
      setMetrics((current) => {
        if (current[page.id]) {
          return current;
        }
        return {
          ...current,
          [page.id]: {
            width: DEFAULT_ASPECT_RATIO,
            height: 1,
          },
        };
      });
    };
  };

  const pageLayouts = useMemo(() => {
    const heights = pages.map((page) => {
      const metric = metrics[page.id];
      const ratio = metric ? metric.width / metric.height : DEFAULT_ASPECT_RATIO;
      return contentWidth / ratio;
    });

    const offsets: number[] = [];
    let cursor = PAGE_PADDING;
    for (const height of heights) {
      offsets.push(cursor);
      cursor += height + PAGE_GAP;
    }

    return {
      heights,
      offsets,
      totalHeight: Math.max(cursor + PAGE_PADDING, contentHeight),
    };
  }, [contentHeight, contentWidth, metrics, pages]);

  const effectiveIndex = useMemo(() => {
    if (mode === "paged") {
      return currentIndex;
    }

    const midpoint = scrollTop + contentHeight / 2;
    let matchIndex = 0;
    for (let index = 0; index < pageLayouts.offsets.length; index += 1) {
      const start = pageLayouts.offsets[index];
      const end = start + pageLayouts.heights[index];
      if (midpoint >= start && midpoint <= end) {
        matchIndex = index;
        break;
      }
      if (midpoint > end) {
        matchIndex = index;
      }
    }
    return matchIndex;
  }, [contentHeight, currentIndex, mode, pageLayouts.heights, pageLayouts.offsets, scrollTop]);

  useEffect(() => {
    if (mode === "scroll") {
      setCurrentIndex(effectiveIndex);
    }
  }, [effectiveIndex, mode]);

  useEffect(() => {
    const start = Math.max(0, effectiveIndex - 5);
    const end = Math.min(pages.length - 1, effectiveIndex + 5);

    for (let index = start; index <= end; index += 1) {
      requestMetric(pages[index]);
    }
  }, [effectiveIndex, pages, metrics]);

  const visibleIndexes = useMemo(() => {
    if (mode !== "scroll") {
      return [];
    }

    const start = Math.max(0, effectiveIndex - 2);
    const end = Math.min(pages.length - 1, effectiveIndex + 2);
    const indexes: number[] = [];
    for (let index = start; index <= end; index += 1) {
      indexes.push(index);
    }
    return indexes;
  }, [effectiveIndex, mode, pages.length]);

  const nextPage = pages[currentIndex + 1];
  const activePage = pages[currentIndex];

  const canDoublePage = useMemo(() => {
    if (mode !== "paged" || !activePage || !nextPage || activePage.chapterID !== nextPage.chapterID) {
      return false;
    }

    const activeMetric = metrics[activePage.id];
    const nextMetric = metrics[nextPage.id];
    const firstRatio = activeMetric ? activeMetric.width / activeMetric.height : DEFAULT_ASPECT_RATIO;
    const secondRatio = nextMetric ? nextMetric.width / nextMetric.height : DEFAULT_ASPECT_RATIO;
    const estimatedWidth = contentHeight * firstRatio + contentHeight * secondRatio;

    return estimatedWidth < contentWidth * 0.98;
  }, [activePage, contentHeight, contentWidth, metrics, mode, nextPage]);

  const pageStep = canDoublePage ? 2 : 1;

  const jumpToIndex = (index: number) => {
    if (pages.length === 0) {
      return;
    }

    const nextIndex = Math.max(0, Math.min(pages.length - 1, index));
    setCurrentIndex(nextIndex);

    if (mode === "scroll") {
      const nextScrollTop = pageLayouts.offsets[nextIndex] ?? 0;
      setScrollTop(nextScrollTop);
      scrollRef.current?.scrollTo({ top: nextScrollTop, behavior: "smooth" });
      scrollRef.current?.focus();
      return;
    }

    containerRef.current?.focus();
  };

  useEffect(() => {
    if (!jumpRequest || pages.length === 0) {
      return;
    }

    const targetIndex =
      jumpRequest.target === "chapter" ? pages.findIndex((page) => page.chapterID === jumpRequest.chapterID) : jumpRequest.page - 1;
    if (targetIndex < 0) {
      pendingJumpIndexRef.current = null;
      return;
    }

    const hasPendingRetry =
      handledJumpRequestIDRef.current === jumpRequest.requestID && pendingJumpIndexRef.current === targetIndex;
    if (!hasPendingRetry && handledJumpRequestIDRef.current === jumpRequest.requestID) {
      return;
    }

    handledJumpRequestIDRef.current = jumpRequest.requestID;

    if (mode === "scroll") {
      for (let index = 0; index <= targetIndex; index += 1) {
        requestMetric(pages[index]);
      }

      const hasAccurateOffsets = pages.slice(0, targetIndex + 1).every((page) => Boolean(metrics[page.id]));
      if (!hasAccurateOffsets) {
        pendingJumpIndexRef.current = targetIndex;
        return;
      }
    }

    pendingJumpIndexRef.current = null;
    jumpToIndex(targetIndex);
  }, [jumpRequest, metrics, mode, pageLayouts.offsets, pages]);

  const handlePagedWheel = (event: WheelEvent<HTMLDivElement>) => {
    if (mode !== "paged") {
      return;
    }

    if (Math.abs(event.deltaY) < 8) {
      return;
    }

    event.preventDefault();
    jumpToIndex(currentIndex + (event.deltaY > 0 ? pageStep : -pageStep));
  };

  const handleKeyDown = (event: KeyboardEvent<HTMLDivElement>) => {
    if (mode === "scroll") {
      if (!scrollRef.current) {
        return;
      }

      if (event.key === "ArrowDown") {
        event.preventDefault();
        scrollRef.current.scrollBy({ top: contentHeight * 0.85, behavior: "smooth" });
      }
      if (event.key === "ArrowUp") {
        event.preventDefault();
        scrollRef.current.scrollBy({ top: -contentHeight * 0.85, behavior: "smooth" });
      }
      return;
    }

    if (event.key === "ArrowRight" || event.key === "ArrowDown" || event.key === " ") {
      event.preventDefault();
      jumpToIndex(currentIndex + pageStep);
    }
    if (event.key === "ArrowLeft" || event.key === "ArrowUp") {
      event.preventDefault();
      jumpToIndex(currentIndex - pageStep);
    }
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
      <div
        ref={containerRef}
        className="h-full min-h-0 overflow-hidden border-l border-border/40 bg-[linear-gradient(180deg,rgba(255,255,255,0.10),rgba(255,255,255,0.02))] backdrop-blur-xl"
      >
        <div
          ref={scrollRef}
          className="h-full overflow-y-auto outline-none"
          onKeyDown={handleKeyDown}
          onScroll={(event) => setScrollTop(event.currentTarget.scrollTop)}
          tabIndex={0}
        >
          <div className="relative" style={{ height: pageLayouts.totalHeight }}>
            {visibleIndexes.map((index) => {
              const page = pages[index];
              const previousPage = pages[index - 1];
              const chapterChanged = !previousPage || previousPage.chapterID !== page.chapterID;

              return (
                <div
                  key={page.id}
                  className="absolute left-1/2 overflow-hidden"
                  style={{
                    top: pageLayouts.offsets[index],
                    width: contentWidth,
                    transform: "translateX(-50%)",
                  }}
                >
                  <div className="relative">
                    {chapterChanged ? (
                      <div className="pointer-events-none absolute left-3 top-3 z-10 inline-flex rounded-full border border-white/70 bg-black/38 px-2.5 py-1 text-[10px] font-semibold uppercase tracking-[0.18em] text-white/90 backdrop-blur-md">
                        {page.chapterTitle}
                      </div>
                    ) : null}
                    <img
                      alt={`${page.chapterTitle} #${page.pageIndex + 1}`}
                      className="block w-full select-none bg-transparent object-contain"
                      draggable={false}
                      loading="eager"
                      onLoad={(event) => {
                        const image = event.currentTarget;
                        if (!image.naturalWidth || !image.naturalHeight) {
                          return;
                        }
                        setMetrics((current) => {
                          if (current[page.id]) {
                            return current;
                          }
                          return {
                            ...current,
                            [page.id]: {
                              width: image.naturalWidth,
                              height: image.naturalHeight,
                            },
                          };
                        });
                      }}
                      src={page.sourceURL}
                    />
                  </div>
                </div>
              );
            })}
          </div>
        </div>
      </div>
    );
  }

  const pagedPages = activePage ? [activePage, ...(canDoublePage && nextPage ? [nextPage] : [])] : [];
  const maxPageWidth = canDoublePage ? contentWidth / 2 : contentWidth;

  return (
    <div
      ref={containerRef}
      className="flex h-full min-h-0 overflow-hidden border-l border-border/40 bg-[linear-gradient(180deg,rgba(255,255,255,0.10),rgba(255,255,255,0.02))] outline-none backdrop-blur-xl"
      onKeyDown={handleKeyDown}
      onWheel={handlePagedWheel}
      tabIndex={0}
    >
      <div className="flex min-h-0 flex-1 items-center justify-center overflow-hidden">
        {pagedPages.map((page) => (
          <img
            key={page.id}
            alt={`${page.chapterTitle} #${page.pageIndex + 1}`}
            className="max-h-full max-w-full select-none object-contain"
            draggable={false}
            loading="eager"
            onLoad={(event) => {
              const image = event.currentTarget;
              if (!image.naturalWidth || !image.naturalHeight) {
                return;
              }
              setMetrics((current) => {
                if (current[page.id]) {
                  return current;
                }
                return {
                  ...current,
                  [page.id]: {
                    width: image.naturalWidth,
                    height: image.naturalHeight,
                  },
                };
              });
            }}
            src={page.sourceURL}
            style={{ height: contentHeight, maxWidth: maxPageWidth }}
          />
        ))}
      </div>
    </div>
  );
}
