import type { PropsWithChildren } from "react";
import { cn } from "@/lib/utils";

const colorMap: Record<string, string> = {
  not_downloaded: "bg-muted text-muted-foreground",
  partial: "bg-warning/15 text-warning",
  complete: "bg-success/15 text-success",
  missing: "bg-danger/15 text-danger",
  running: "bg-primary/15 text-primary",
  paused: "bg-warning/15 text-warning",
  completed: "bg-success/15 text-success",
  failed: "bg-danger/15 text-danger",
  queued: "bg-muted text-muted-foreground",
};

export function Badge({ children, className, tone = "default" }: PropsWithChildren<{ className?: string; tone?: string }>) {
  return (
    <span
      className={cn(
        "inline-flex items-center rounded-full px-2.5 py-1 text-[11px] font-semibold uppercase tracking-[0.14em]",
        colorMap[tone] ?? "bg-primary/10 text-primary",
        className
      )}
    >
      {children}
    </span>
  );
}

