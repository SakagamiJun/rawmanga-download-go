import { cn } from "@/lib/utils";

export function Progress({ value, className }: { value: number; className?: string }) {
  const bounded = Math.max(0, Math.min(100, value));
  return (
    <div className={cn("h-2.5 w-full overflow-hidden rounded-full bg-primary/10", className)}>
      <div
        className="h-full rounded-full bg-gradient-to-r from-primary via-sky-500 to-success transition-[width]"
        style={{ width: `${bounded}%` }}
      />
    </div>
  );
}

