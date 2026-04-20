import { type KeyboardEvent, type WheelEvent, useEffect, useMemo, useRef, useState } from "react";
import { type FlatReaderPage, type PageMetric, type ReaderNavigationRequest, DEFAULT_ASPECT_RATIO, PAGE_PADDING, clampIndex } from "@/components/reader-shared";

interface PagedReaderProps {
  currentIndex: number;
  metrics: Record<string, PageMetric>;
  navigationRequest: ReaderNavigationRequest | null;
  onCurrentIndexChange: (index: number) => void;
  onMetricMeasured: (pageID: string, width: number, height: number) => void;
  pages: FlatReaderPage[];
  requestMetric: (page: FlatReaderPage | undefined) => void;
}

export function PagedReader({
  currentIndex,
  metrics,
  navigationRequest,
  onCurrentIndexChange,
  onMetricMeasured,
  pages,
  requestMetric,
}: PagedReaderProps) {
  const containerRef = useRef<HTMLDivElement | null>(null);
  const handledNavigationIDRef = useRef<number | null>(null);

  const [viewportWidth, setViewportWidth] = useState(0);
  const [viewportHeight, setViewportHeight] = useState(0);

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
  }, [pages]);

  useEffect(() => {
    if (!navigationRequest || handledNavigationIDRef.current === navigationRequest.id) {
      return;
    }

    handledNavigationIDRef.current = navigationRequest.id;
    containerRef.current?.focus();
  }, [navigationRequest]);

  const contentWidth = Math.max(320, viewportWidth - PAGE_PADDING * 2);
  const contentHeight = Math.max(320, viewportHeight - PAGE_PADDING * 2);

  const activePage = pages[currentIndex];
  const nextPage = pages[currentIndex + 1];

  useEffect(() => {
    requestMetric(activePage);
    requestMetric(nextPage);
    requestMetric(pages[currentIndex - 1]);
  }, [activePage, currentIndex, nextPage, pages, requestMetric]);

  const canDoublePage = useMemo(() => {
    if (!activePage || !nextPage || activePage.chapterID !== nextPage.chapterID) {
      return false;
    }

    const activeMetric = metrics[activePage.id];
    const nextMetric = metrics[nextPage.id];
    const firstRatio = activeMetric ? activeMetric.width / activeMetric.height : DEFAULT_ASPECT_RATIO;
    const secondRatio = nextMetric ? nextMetric.width / nextMetric.height : DEFAULT_ASPECT_RATIO;
    const estimatedWidth = contentHeight * firstRatio + contentHeight * secondRatio;

    return estimatedWidth < contentWidth * 0.98;
  }, [activePage, contentHeight, contentWidth, metrics, nextPage]);

  const pageStep = canDoublePage ? 2 : 1;

  const jumpToIndex = (index: number) => {
    onCurrentIndexChange(clampIndex(index, pages.length));
  };

  const handleWheel = (event: WheelEvent<HTMLDivElement>) => {
    if (Math.abs(event.deltaY) < 8) {
      return;
    }

    event.preventDefault();
    jumpToIndex(currentIndex + (event.deltaY > 0 ? pageStep : -pageStep));
  };

  const handleKeyDown = (event: KeyboardEvent<HTMLDivElement>) => {
    if (event.key === "ArrowRight" || event.key === "ArrowDown" || event.key === " ") {
      event.preventDefault();
      jumpToIndex(currentIndex + pageStep);
    }
    if (event.key === "ArrowLeft" || event.key === "ArrowUp") {
      event.preventDefault();
      jumpToIndex(currentIndex - pageStep);
    }
  };

  const pagedPages = activePage ? [activePage, ...(canDoublePage && nextPage ? [nextPage] : [])] : [];
  const maxPageWidth = canDoublePage ? contentWidth / 2 : contentWidth;

  return (
    <div
      ref={containerRef}
      className="flex h-full min-h-0 overflow-hidden border-l border-border/40 bg-[linear-gradient(180deg,rgba(255,255,255,0.10),rgba(255,255,255,0.02))] outline-none backdrop-blur-xl"
      onKeyDown={handleKeyDown}
      onWheel={handleWheel}
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
              onMetricMeasured(page.id, image.naturalWidth, image.naturalHeight);
            }}
            src={page.sourceURL}
            style={{ height: contentHeight, maxWidth: maxPageWidth }}
          />
        ))}
      </div>
    </div>
  );
}
