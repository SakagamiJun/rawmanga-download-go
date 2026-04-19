export function getWailsApp() {
  if (typeof window === "undefined") {
    return null;
  }

  return (window as Window & {
    go?: {
      main?: {
        App?: Record<string, (...args: unknown[]) => Promise<unknown>>;
      };
    };
  }).go?.main?.App ?? null;
}

export function getWailsRuntime() {
  if (typeof window === "undefined") {
    return null;
  }

  return (window as Window & {
    runtime?: {
      EventsOn?: (eventName: string, callback: (payload: unknown) => void) => void | (() => void);
      EventsOff?: (eventName: string) => void;
      EventsEmit?: (eventName: string, payload: unknown) => void;
    };
  }).runtime ?? null;
}

export function emitRuntimeEvent(eventName: string, payload: unknown) {
  getWailsRuntime()?.EventsEmit?.(eventName, payload);
}

