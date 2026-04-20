import { type KeyboardEvent, useEffect, useLayoutEffect, useMemo, useRef, useState } from "react";
import { type FlatReaderPage, type PageMetric, type ReaderNavigationRequest, DEFAULT_ASPECT_RATIO, PAGE_GAP, PAGE_PADDING, clampIndex, findPageIndexAtPosition } from "@/components/reader-shared";

interface ScrollReaderProps {
  cacheRadius: number;
  currentIndex: number;
  metrics: Record<string, PageMetric>;
  navigationRequest: ReaderNavigationRequest | null;
  onCurrentIndexChange: (index: number) => void;
  onMetricMeasured: (pageID: string, width: number, height: number) => void;
  pages: FlatReaderPage[];
  requestMetric: (page: FlatReaderPage | undefined) => void;
}

export function ScrollReader({
  cacheRadius,
  currentIndex,
  metrics,
  navigationRequest,
  onCurrentIndexChange,
  onMetricMeasured,
  pages,
  requestMetric,
}: ScrollReaderProps) {
  const containerRef = useRef<HTMLDivElement | null>(null);
  const scrollRef = useRef<HTMLDivElement | null>(null);
  const handledNavigationIDRef = useRef<number | null>(null);
  const pendingNavigationIndexRef = useRef<number | null>(null);

  const [viewportWidth, setViewportWidth] = useState(0);
  const [viewportHeight, setViewportHeight] = useState(0);
  const [scrollTop, setScrollTop] = useState(0);

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

  useEffect(() => {
    handledNavigationIDRef.current = null;
    pendingNavigationIndexRef.current = null;
    setScrollTop(0);
    if (scrollRef.current) {
      scrollRef.current.scrollTop = 0;
    }
  }, [pages]);

  const contentWidth = Math.max(320, viewportWidth - PAGE_PADDING * 2);
  const contentHeight = Math.max(320, viewportHeight - PAGE_PADDING * 2);

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
    return findPageIndexAtPosition(pageLayouts.offsets, pageLayouts.heights, scrollTop + contentHeight / 2);
  }, [contentHeight, pageLayouts.heights, pageLayouts.offsets, scrollTop]);

  useEffect(() => {
    if (navigationRequest && handledNavigationIDRef.current !== navigationRequest.id) {
      return;
    }

    const pendingNavigationIndex = pendingNavigationIndexRef.current;
    if (pendingNavigationIndex !== null && effectiveIndex !== pendingNavigationIndex) {
      return;
    }

    if (pendingNavigationIndex === effectiveIndex) {
      pendingNavigationIndexRef.current = null;
    }

    onCurrentIndexChange(effectiveIndex);
  }, [effectiveIndex, navigationRequest, onCurrentIndexChange]);

  const firstVisibleIndex = useMemo(() => {
    return findPageIndexAtPosition(pageLayouts.offsets, pageLayouts.heights, scrollTop);
  }, [pageLayouts.heights, pageLayouts.offsets, scrollTop]);

  const lastVisibleIndex = useMemo(() => {
    return findPageIndexAtPosition(pageLayouts.offsets, pageLayouts.heights, scrollTop + contentHeight);
  }, [contentHeight, pageLayouts.heights, pageLayouts.offsets, scrollTop]);

  const renderRange = useMemo(() => {
    const start = Math.max(0, firstVisibleIndex - cacheRadius);
    const end = Math.min(pages.length - 1, lastVisibleIndex + cacheRadius);
    return { start, end };
  }, [cacheRadius, firstVisibleIndex, lastVisibleIndex, pages.length]);

  useEffect(() => {
    for (let index = renderRange.start; index <= renderRange.end; index += 1) {
      requestMetric(pages[index]);
    }
  }, [pages, renderRange.end, renderRange.start, requestMetric]);

  useLayoutEffect(() => {
    if (!navigationRequest || pages.length === 0) {
      return;
    }

    const targetIndex = clampIndex(navigationRequest.index, pages.length);
    const hasPendingRetry =
      handledNavigationIDRef.current === navigationRequest.id && pendingNavigationIndexRef.current === targetIndex;
    if (!hasPendingRetry && handledNavigationIDRef.current === navigationRequest.id) {
      return;
    }

    handledNavigationIDRef.current = navigationRequest.id;

    for (let index = 0; index <= targetIndex; index += 1) {
      requestMetric(pages[index]);
    }

    const hasAccurateOffsets = pages.slice(0, targetIndex + 1).every((page) => Boolean(metrics[page.id]));
    if (!hasAccurateOffsets) {
      pendingNavigationIndexRef.current = targetIndex;
      return;
    }

    pendingNavigationIndexRef.current = null;

    const nextScrollTop = pageLayouts.offsets[targetIndex] ?? 0;
    setScrollTop(nextScrollTop);
    scrollRef.current?.scrollTo({
      top: nextScrollTop,
      behavior: navigationRequest.reason === "jump" ? "smooth" : "auto",
    });
    scrollRef.current?.focus();
  }, [metrics, navigationRequest, pageLayouts.offsets, pages, requestMetric]);

  const handleKeyDown = (event: KeyboardEvent<HTMLDivElement>) => {
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
  };

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
          {pages.slice(renderRange.start, renderRange.end + 1).map((page, offsetIndex) => {
            const index = renderRange.start + offsetIndex;
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
                    loading={Math.abs(index - currentIndex) <= 1 ? "eager" : "lazy"}
                    onLoad={(event) => {
                      const image = event.currentTarget;
                      if (!image.naturalWidth || !image.naturalHeight) {
                        return;
                      }
                      onMetricMeasured(page.id, image.naturalWidth, image.naturalHeight);
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
