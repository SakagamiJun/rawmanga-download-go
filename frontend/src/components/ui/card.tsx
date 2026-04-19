import type { PropsWithChildren } from "react";
import { cn } from "@/lib/utils";

export function Card({ children, className }: PropsWithChildren<{ className?: string }>) {
  return (
    <section
      className={cn(
        "rounded-3xl border border-border/70 bg-card/90 p-5 shadow-[0_18px_60px_rgba(18,39,74,0.08)] backdrop-blur-sm dark:shadow-[0_18px_60px_rgba(0,0,0,0.28)]",
        className
      )}
    >
      {children}
    </section>
  );
}

