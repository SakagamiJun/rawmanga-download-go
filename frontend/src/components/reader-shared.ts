import type { ReaderPage } from "@/lib/contracts";

export interface PageMetric {
  width: number;
  height: number;
}

export interface FlatReaderPage extends ReaderPage {
  globalIndex: number;
  globalPage: number;
}

export interface ReaderNavigationRequest {
  id: number;
  index: number;
  reason: "jump" | "restore" | "sync";
}

export const DEFAULT_ASPECT_RATIO = 0.72;
export const PAGE_GAP = 0;
export const PAGE_PADDING = 0;

export function clampIndex(index: number, total: number) {
  if (total <= 0) {
    return 0;
  }

  return Math.max(0, Math.min(total - 1, index));
}

export function findPageIndexAtPosition(offsets: number[], heights: number[], position: number) {
  if (offsets.length === 0) {
    return 0;
  }

  let matchIndex = 0;
  for (let index = 0; index < offsets.length; index += 1) {
    const start = offsets[index];
    const end = start + heights[index];
    if (position >= start && position <= end) {
      return index;
    }
    if (position > end) {
      matchIndex = index;
    }
  }

  return matchIndex;
}
