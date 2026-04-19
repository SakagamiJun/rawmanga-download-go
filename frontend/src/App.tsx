import { type ReactNode, useEffect, useMemo, useRef, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import {
  ArrowLeft,
  BookImage,
  ChevronsLeft,
  ChevronsRight,
  Download,
  Languages,
  MoonStar,
  RefreshCcw,
  Search,
  Settings2,
  Sparkles,
  SunMedium,
  Telescope,
} from "lucide-react";
import { ReaderView, type ReaderJumpRequest } from "@/components/reader-view";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { Progress } from "@/components/ui/progress";
import { Select } from "@/components/ui/select";
import { appAdapter } from "@/lib/api";
import {
  type AppSettings,
  type ChapterItem,
  EVENTS,
  type LibraryManga,
  type LocalChapterStatus,
  type DownloadJob,
  type DownloadProgressEvent,
} from "@/lib/contracts";
import { i18n } from "@/lib/i18n";
import { emitRuntimeEvent } from "@/lib/runtime";
import { resolveLocale, resolveTheme } from "@/lib/system";
import { cn, formatBytesPerSecond, formatDateTime } from "@/lib/utils";
import { useDownloadStore } from "@/store/download-store";

type PaneMode = "tasks" | "settings";
type ReaderMode = "scroll" | "paged";

const statusOptions: Array<LocalChapterStatus | "all"> = ["all", "not_downloaded", "partial", "complete", "missing"];
const floatingSurfaceClass =
  "border border-slate-200/80 bg-[rgba(236,241,246,0.84)] text-slate-800 shadow-[0_1px_0_rgba(255,255,255,0.62)_inset,0_14px_36px_rgba(15,23,42,0.16)] backdrop-blur-2xl supports-[backdrop-filter]:bg-[rgba(236,241,246,0.72)]";

function formatLocaleState(settings: AppSettings | undefined, t: (key: string) => string) {
  if (!settings || settings.localeMode === "system") {
    return t("settings.system");
  }

  switch (settings.locale) {
    case "zh-CN":
      return "中文";
    case "ja":
      return "日本語";
    default:
      return "English";
  }
}

function formatThemeState(settings: AppSettings | undefined, t: (key: string) => string) {
  if (!settings) {
    return t("settings.system");
  }

  switch (settings.themeMode) {
    case "light":
      return t("settings.light");
    case "dark":
      return t("settings.dark");
    default:
      return t("settings.system");
  }
}

export default function App() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();

  const [sourceURL, setSourceURL] = useState("https://klz9.com/otona-ni-narenai-bokura-wa.html");
  const [parsed, setParsed] = useState<null | Awaited<ReturnType<typeof appAdapter.resolveManga>>>(null);
  const [selectedChapterIDs, setSelectedChapterIDs] = useState<string[]>([]);
  const [statusFilter, setStatusFilter] = useState<LocalChapterStatus | "all">("all");
  const [paneMode, setPaneMode] = useState<PaneMode>("tasks");
  const [paneVisible, setPaneVisible] = useState(false);
  const [paneWidth, setPaneWidth] = useState(380);
  const [selectedLibraryID, setSelectedLibraryID] = useState<string | null>(null);
  const [readerMode, setReaderMode] = useState<ReaderMode>("scroll");
  const [readerJumpMenuOpen, setReaderJumpMenuOpen] = useState(false);
  const [readerJumpChapterID, setReaderJumpChapterID] = useState("");
  const [readerJumpPageInput, setReaderJumpPageInput] = useState("");
  const [readerJumpRequest, setReaderJumpRequest] = useState<ReaderJumpRequest | null>(null);

  const readerJumpPanelRef = useRef<HTMLDivElement | null>(null);

  const jobsByID = useDownloadStore((state) => state.jobs);
  const progressMap = useDownloadStore((state) => state.progress);
  const seedJobs = useDownloadStore((state) => state.seedJobs);
  const mergeJob = useDownloadStore((state) => state.mergeJob);
  const mergeProgress = useDownloadStore((state) => state.mergeProgress);

  const settingsQuery = useQuery({
    queryKey: ["settings"],
    queryFn: () => appAdapter.getSettings(),
  });

  const jobsQuery = useQuery({
    queryKey: ["jobs"],
    queryFn: () => appAdapter.listDownloadJobs(),
  });

  const libraryQuery = useQuery({
    queryKey: ["library"],
    queryFn: () => appAdapter.listLibraryManga(),
  });

  const readerQuery = useQuery({
    queryKey: ["reader", selectedLibraryID],
    enabled: Boolean(selectedLibraryID),
    queryFn: () => appAdapter.getReaderManifest(selectedLibraryID!),
  });

  useEffect(() => {
    if (jobsQuery.data) {
      seedJobs(jobsQuery.data);
    }
  }, [jobsQuery.data, seedJobs]);

  useEffect(() => {
    const offJob = appAdapter.subscribe(EVENTS.DOWNLOAD_JOB, (payload) => {
      mergeJob(payload as DownloadJob);
      void queryClient.invalidateQueries({ queryKey: ["jobs"] });
    });

    const offChapter = appAdapter.subscribe(EVENTS.DOWNLOAD_CHAPTER, (payload) => {
      const event = payload as DownloadProgressEvent;
      mergeProgress(event);
      if (event.status === "completed" || event.status === "failed") {
        void queryClient.invalidateQueries({ queryKey: ["library"] });
      }
    });

    const offPage = appAdapter.subscribe(EVENTS.DOWNLOAD_PAGE, (payload) => {
      mergeProgress(payload as DownloadProgressEvent);
    });

    const offLibrary = appAdapter.subscribe(EVENTS.LIBRARY_RECONCILED, () => {
      void queryClient.invalidateQueries({ queryKey: ["library"] });
    });

    return () => {
      offJob();
      offChapter();
      offPage();
      offLibrary();
    };
  }, [mergeJob, mergeProgress, queryClient]);

  useEffect(() => {
    const settings = settingsQuery.data;
    if (!settings) {
      return;
    }

    const mediaQuery = typeof window.matchMedia === "function" ? window.matchMedia("(prefers-color-scheme: dark)") : null;
    const applyTheme = () => {
      const resolvedTheme = resolveTheme(settings.themeMode, mediaQuery?.matches ?? false);
      document.documentElement.setAttribute("data-theme", resolvedTheme);
      emitRuntimeEvent(EVENTS.THEME_RESOLVED, {
        mode: settings.themeMode,
        resolved: resolvedTheme,
      });
    };

    const resolvedLocale = resolveLocale(settings, navigator.languages);
    void i18n.changeLanguage(resolvedLocale);
    emitRuntimeEvent(EVENTS.LOCALE_RESOLVED, {
      mode: settings.localeMode,
      locale: resolvedLocale,
    });

    applyTheme();
    if (!mediaQuery) {
      return;
    }

    if (typeof mediaQuery.addEventListener === "function") {
      mediaQuery.addEventListener("change", applyTheme);
      return () => mediaQuery.removeEventListener("change", applyTheme);
    }

    if (typeof mediaQuery.addListener === "function") {
      mediaQuery.addListener(applyTheme);
      return () => mediaQuery.removeListener(applyTheme);
    }
  }, [settingsQuery.data]);

  useEffect(() => {
    const chapters = readerQuery.data?.chapters ?? [];
    if (chapters.length === 0) {
      setReaderJumpChapterID("");
      setReaderJumpPageInput("");
      setReaderJumpMenuOpen(false);
      return;
    }

    setReaderJumpChapterID((current) => {
      if (chapters.some((chapter) => chapter.id === current)) {
        return current;
      }
      return chapters[0]?.id ?? "";
    });
  }, [readerQuery.data]);

  useEffect(() => {
    setReaderJumpMenuOpen(false);
    setReaderJumpPageInput("");
    setReaderJumpRequest(null);
  }, [selectedLibraryID]);

  useEffect(() => {
    if (!readerJumpMenuOpen) {
      return;
    }

    const handlePointerDown = (event: MouseEvent) => {
      if (readerJumpPanelRef.current?.contains(event.target as Node)) {
        return;
      }
      setReaderJumpMenuOpen(false);
    };

    window.addEventListener("mousedown", handlePointerDown);
    return () => window.removeEventListener("mousedown", handlePointerDown);
  }, [readerJumpMenuOpen]);

  const resolveMutation = useMutation({
    mutationFn: (url: string) => appAdapter.resolveManga(url),
    onSuccess: (result) => {
      setParsed(result);
      setSelectedChapterIDs(result.chapters.map((chapter) => chapter.id));
      setPaneMode("tasks");
      setPaneVisible(true);
    },
  });

  const queueMutation = useMutation({
    mutationFn: (request: { settings: AppSettings; chapters: ChapterItem[] }) =>
      appAdapter.queueChapters({
        sourceURL: parsed?.sourceURL ?? sourceURL,
        mangaSlug: parsed?.slug ?? "",
        title: parsed?.title ?? "",
        chapterIDs: request.chapters.map((chapter) => chapter.id),
        outputRoot: request.settings.outputRoot,
      }),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["jobs"] });
    },
  });

  const settingsMutation = useMutation({
    mutationFn: (input: AppSettings) => appAdapter.updateSettings(input),
    onSuccess: async (updated) => {
      queryClient.setQueryData(["settings"], updated);
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["settings"] }),
        queryClient.invalidateQueries({ queryKey: ["library"] }),
      ]);
    },
  });

  const settings = settingsQuery.data;

  const filteredChapters = useMemo(() => {
    if (!parsed) {
      return [];
    }

    return parsed.chapters.filter((chapter) => statusFilter === "all" || chapter.localStatus === statusFilter);
  }, [parsed, statusFilter]);

  const jobs = useMemo(
    () =>
      Object.values(jobsByID).sort((left, right) => {
        return right.updatedAt.localeCompare(left.updatedAt);
      }),
    [jobsByID]
  );

  const progressByJob = useMemo(() => {
    return jobs.reduce<Record<string, DownloadProgressEvent | undefined>>((accumulator, job) => {
      const candidates = Object.entries(progressMap)
        .filter(([key]) => key.startsWith(`${job.jobID}:`))
        .map(([, event]) => event);
      accumulator[job.jobID] = candidates.sort((left, right) => right.pageIndex - left.pageIndex)[0];
      return accumulator;
    }, {});
  }, [jobs, progressMap]);

  const selectedChapterCount = selectedChapterIDs.length;
  const canStartProcessing = Boolean(settings && parsed && selectedChapterCount > 0 && !queueMutation.isPending);
  const library = libraryQuery.data ?? [];
  const selectedLibrary = library.find((item) => item.id === selectedLibraryID) ?? null;
  const readerManifest = readerQuery.data ?? null;
  const manualReaderPage = Number(readerJumpPageInput);
  const canJumpToReaderPage =
    Number.isInteger(manualReaderPage) && manualReaderPage >= 1 && manualReaderPage <= (readerManifest?.totalPages ?? 0);
  const localeState = formatLocaleState(settings, t);
  const localeBadge = formatLocaleBadge(settings);
  const themeState = formatThemeState(settings, t);
  const themeIcon =
    settings?.themeMode === "dark" ? (
      <MoonStar className="h-4 w-4" />
    ) : settings?.themeMode === "light" ? (
      <SunMedium className="h-4 w-4" />
    ) : (
      <Sparkles className="h-4 w-4" />
    );

  const togglePane = (nextPane: PaneMode) => {
    setPaneVisible((current) => (paneMode === nextPane ? !current : true));
    setPaneMode(nextPane);
  };

  const startResize = (clientX: number, initialWidth: number) => {
    const handleMouseMove = (event: MouseEvent) => {
      const nextWidth = initialWidth + event.clientX - clientX;
      setPaneWidth(Math.min(500, Math.max(340, nextWidth)));
    };

    const handleMouseUp = () => {
      window.removeEventListener("mousemove", handleMouseMove);
      window.removeEventListener("mouseup", handleMouseUp);
    };

    window.addEventListener("mousemove", handleMouseMove);
    window.addEventListener("mouseup", handleMouseUp);
  };

  const submitSelectedChapters = () => {
    if (!settings || !parsed) {
      return;
    }

    queueMutation.mutate({
      settings,
      chapters: parsed.chapters.filter((chapter) => selectedChapterIDs.includes(chapter.id)),
    });
  };

  const submitReaderChapterJump = () => {
    if (!readerJumpChapterID) {
      return;
    }

    setReaderJumpRequest({
      requestID: Date.now(),
      target: "chapter",
      chapterID: readerJumpChapterID,
    });
    setReaderJumpMenuOpen(false);
  };

  const submitReaderPageJump = () => {
    if (!canJumpToReaderPage) {
      return;
    }

    setReaderJumpRequest({
      requestID: Date.now(),
      target: "page",
      page: manualReaderPage,
    });
    setReaderJumpMenuOpen(false);
  };

  const cycleTheme = () => {
    if (!settings) {
      return;
    }

    const nextTheme =
      settings.themeMode === "system" ? "light" : settings.themeMode === "light" ? "dark" : "system";

    settingsMutation.mutate({
      ...settings,
      themeMode: nextTheme,
    });
  };

  const cycleLocale = () => {
    if (!settings) {
      return;
    }

    const sequence: Array<{ localeMode: AppSettings["localeMode"]; locale: AppSettings["locale"] }> = [
      { localeMode: "system", locale: "en" },
      { localeMode: "manual", locale: "zh-CN" },
      { localeMode: "manual", locale: "en" },
      { localeMode: "manual", locale: "ja" },
    ];

    const currentIndex = sequence.findIndex(
      (item) => item.localeMode === settings.localeMode && (settings.localeMode === "system" || item.locale === settings.locale)
    );
    const next = sequence[(currentIndex + 1 + sequence.length) % sequence.length];

    settingsMutation.mutate({
      ...settings,
      localeMode: next.localeMode,
      locale: next.locale,
    });
  };

  return (
    <main className="h-screen overflow-hidden bg-[linear-gradient(180deg,rgba(255,255,255,0.08),rgba(255,255,255,0)),radial-gradient(circle_at_top_left,rgba(116,162,255,0.10),transparent_24%),linear-gradient(180deg,hsl(var(--background)),hsl(var(--background)))] text-foreground">
      <div className="flex h-full border border-border/60">
        <aside className="flex h-full w-[72px] shrink-0 flex-col items-center justify-between border-r border-border/60 bg-[var(--app-sidebar)] px-3 py-4 backdrop-blur-xl">
          <div className="mt-5 flex flex-col items-center gap-4">
            <div className="flex h-10 w-10 items-center justify-center rounded-2xl bg-foreground text-background text-sm font-black tracking-[0.12em] shadow-[0_12px_30px_rgba(15,23,42,0.16)]">
              KLZ
            </div>

            <div className="space-y-2">
              <RailIconButton
                active={paneVisible && paneMode === "tasks"}
                icon={<Download className="h-4 w-4" />}
                label={t("shell.tasks")}
                onClick={() => togglePane("tasks")}
              />
              <RailIconButton
                active={paneVisible && paneMode === "settings"}
                icon={<Settings2 className="h-4 w-4" />}
                label={t("shell.settings")}
                onClick={() => togglePane("settings")}
              />
            </div>
          </div>

          <div className="flex flex-col items-center gap-3">
            <RailUtilityButton
              disabled={!settings || settingsMutation.isPending}
              icon={themeIcon}
              label={t("shell.theme")}
              onClick={cycleTheme}
              title={`${t("shell.theme")}: ${themeState}`}
            />
            <RailUtilityButton
              badge={localeBadge}
              disabled={!settings || settingsMutation.isPending}
              icon={<Languages className="h-4 w-4" />}
              label={t("shell.language")}
              onClick={cycleLocale}
              title={`${t("shell.language")}: ${localeState}`}
            />
            <RailUtilityButton
              icon={paneVisible ? <ChevronsLeft className="h-4 w-4" /> : <ChevronsRight className="h-4 w-4" />}
              label={paneVisible ? t("shell.collapseSidebar") : t("shell.expandSidebar")}
              onClick={() => setPaneVisible((current) => !current)}
              title={paneVisible ? t("shell.collapseSidebar") : t("shell.expandSidebar")}
            />
          </div>
        </aside>

        <div className="relative min-w-0 flex-1">
          <section className="relative h-full overflow-hidden bg-card/20 ">
            <div className="app-window-drag-region absolute inset-x-0 top-0 z-10 flex items-start justify-between gap-3 px-4 py-2">
              <div className={cn("flex max-w-[min(58vw,32rem)] items-center gap-2 px-3 py-1.5 text-xs font-semibold", floatingSurfaceClass)}>
                <span className="truncate">{selectedLibrary ? selectedLibrary.title : t("library.heading")}</span>
                <Badge tone={selectedLibrary ? "running" : appAdapter.mode === "mock" ? "queued" : "completed"}>
                  {selectedLibrary
                    ? `${selectedLibrary.chapterCount} ${t("library.chapters")}`
                    : appAdapter.mode === "mock"
                      ? t("app.mock")
                      : t("app.live")}
                </Badge>
              </div>

              {selectedLibraryID ? (
                <div className="app-window-no-drag flex flex-wrap items-center justify-end gap-2">
                  <Button
                    type="button"
                    size="sm"
                    variant="outline"
                    className={cn(
                      "gap-2 px-3",
                      floatingSurfaceClass,
                      readerMode === "scroll"
                        ? "border-slate-400/70 bg-[rgba(230,236,242,0.96)] text-slate-900"
                        : "text-slate-700 hover:bg-[rgba(236,241,246,0.92)]"
                    )}
                    onClick={() => setReaderMode("scroll")}
                  >
                    {t("reader.scrollMode")}
                  </Button>
                  <Button
                    type="button"
                    size="sm"
                    variant="outline"
                    className={cn(
                      "gap-2 px-3",
                      floatingSurfaceClass,
                      readerMode === "paged"
                        ? "border-slate-400/70 bg-[rgba(230,236,242,0.96)] text-slate-900"
                        : "text-slate-700 hover:bg-[rgba(236,241,246,0.92)]"
                    )}
                    onClick={() => setReaderMode("paged")}
                  >
                    {t("reader.pagedMode")}
                  </Button>
                  <div className="relative" ref={readerJumpPanelRef}>
                    <Button
                      type="button"
                      size="sm"
                      variant="outline"
                      className={cn(
                        "gap-2 px-3 text-slate-800 hover:bg-[rgba(236,241,246,0.92)]",
                        floatingSurfaceClass,
                        readerJumpMenuOpen ? "border-slate-400/70 bg-[rgba(230,236,242,0.96)] text-slate-900" : null
                      )}
                      disabled={!readerManifest}
                      onClick={() => setReaderJumpMenuOpen((current) => !current)}
                    >
                      <Telescope className="h-4 w-4" />
                      {t("reader.jump")}
                    </Button>

                    {readerJumpMenuOpen && readerManifest ? (
                      <div
                        className={cn(
                          "absolute right-0 top-full z-20 mt-2 w-[min(20rem,calc(100vw-2rem))] rounded-2xl p-3 text-left",
                          floatingSurfaceClass
                        )}
                      >
                        <div className="space-y-3">
                          <div className="border-b border-slate-200/80 pb-2">
                            <p className="text-[10px] font-semibold uppercase tracking-[0.22em] text-slate-500">{t("reader.jumpTitle")}</p>
                            <p className="mt-1 text-xs text-slate-600">{t("reader.jumpRange", { total: readerManifest.totalPages })}</p>
                          </div>

                          <form
                            className="space-y-2"
                            onSubmit={(event) => {
                              event.preventDefault();
                              submitReaderChapterJump();
                            }}
                          >
                            <label className="block text-[11px] font-semibold uppercase tracking-[0.16em] text-slate-500">
                              {t("reader.jumpChapterLabel")}
                            </label>
                            <Select
                              className="h-10 rounded-xl border-slate-300/80 bg-white/88 text-sm text-slate-900"
                              onChange={(event) => setReaderJumpChapterID(event.target.value)}
                              value={readerJumpChapterID}
                            >
                              {readerManifest.chapters.map((chapter) => (
                                <option key={chapter.id} value={chapter.id}>
                                  {chapter.number > 0 ? `${chapter.number} · ${chapter.title}` : chapter.title}
                                </option>
                              ))}
                            </Select>
                            <Button className="w-full" size="sm" type="submit" variant="outline">
                              {t("reader.jumpChapterAction")}
                            </Button>
                          </form>

                          <form
                            className="space-y-2 border-t border-slate-200/80 pt-3"
                            onSubmit={(event) => {
                              event.preventDefault();
                              submitReaderPageJump();
                            }}
                          >
                            <label className="block text-[11px] font-semibold uppercase tracking-[0.16em] text-slate-500">
                              {t("reader.jumpPageLabel")}
                            </label>
                            <Input
                              className="h-10 rounded-xl border-slate-300/80 bg-white/88 text-slate-900"
                              inputMode="numeric"
                              max={readerManifest.totalPages}
                              min={1}
                              onChange={(event) => setReaderJumpPageInput(event.target.value)}
                              placeholder={t("reader.jumpPagePlaceholder")}
                              type="number"
                              value={readerJumpPageInput}
                            />
                            <Button className="w-full" disabled={!canJumpToReaderPage} size="sm" type="submit" variant="outline">
                              {t("reader.jumpPageAction")}
                            </Button>
                          </form>
                        </div>
                      </div>
                    ) : null}
                  </div>
                  <Button
                    className={cn("gap-2 px-3 text-slate-800 hover:bg-[rgba(236,241,246,0.92)]", floatingSurfaceClass)}
                    onClick={() => {
                      setReaderJumpMenuOpen(false);
                      setSelectedLibraryID(null);
                    }}
                    size="sm"
                    variant="outline"
                  >
                    <ArrowLeft className="h-4 w-4" />
                    {t("library.back")}
                  </Button>
                </div>
              ) : null}
            </div>

            <div className="h-full pt-0">
              {selectedLibraryID && readerQuery.isLoading ? (
                <div className="flex h-full items-center justify-center border-l border-border/40 bg-card/14 text-sm text-muted-foreground">
                  Loading reader…
                </div>
              ) : selectedLibraryID && readerQuery.isError ? (
                <div className="flex h-full items-center justify-center border-l border-danger/20 bg-card/14 text-sm text-danger">
                  Failed to open this manga reader.
                </div>
              ) : selectedLibraryID && readerQuery.data ? (
                <ReaderView jumpRequest={readerJumpRequest} manifest={readerQuery.data} mode={readerMode} />
              ) : (
                <LibraryGrid
                  emptyLabel={t("library.empty")}
                  items={library}
                  loading={libraryQuery.isLoading}
                  onOpen={setSelectedLibraryID}
                />
              )}
            </div>
          </section>

          {paneVisible ? (
            <div className="pointer-events-none absolute inset-y-0 left-0 z-20 flex" style={{ width: paneWidth + 10 }}>
              <section
                className="pointer-events-auto h-full overflow-hidden border-r border-border/60 bg-card/90 backdrop-blur-xl"
                style={{ width: paneWidth }}
              >
                <div className="flex h-full flex-col">
                  <div className="app-window-drag-region border-b border-border/60 px-4 py-3">
                    <div className="flex items-center justify-between gap-3">
                      <div className="min-w-0">
                        <p className="text-[10px] font-semibold uppercase tracking-[0.24em] text-muted-foreground">{t("shell.context")}</p>
                        <h1 className="mt-1 truncate text-base font-black">
                          {paneMode === "tasks" ? t("shell.processing") : t("settings.title")}
                        </h1>
                      </div>
                      <Badge tone={appAdapter.mode === "mock" ? "queued" : "running"}>
                        {appAdapter.mode === "mock" ? t("app.mock") : t("app.live")}
                      </Badge>
                    </div>
                  </div>

                  <div className="flex-1 overflow-y-auto">
                    {paneMode === "tasks" ? (
                      <div className="space-y-3 p-3">
                        <PanelSection title={t("resolve.title")} subtitle={t("resolve.subtitle")}>
                          <form
                            className="grid gap-2.5"
                            onSubmit={(event) => {
                              event.preventDefault();
                              void resolveMutation.mutateAsync(sourceURL);
                            }}
                          >
                            <Input
                              value={sourceURL}
                              onChange={(event) => setSourceURL(event.target.value)}
                              placeholder={t("resolve.placeholder")}
                            />
                            <Button className="w-full gap-2" disabled={resolveMutation.isPending} type="submit">
                              <Search className="h-4 w-4" />
                              {t("resolve.submit")}
                            </Button>
                          </form>

                          {parsed ? (
                            <div className="mt-2.5 flex flex-wrap gap-2">
                              <Badge tone={parsed.profileCacheHit ? "complete" : "partial"}>
                                {parsed.profileCacheHit ? t("resolve.cacheHit") : t("resolve.cacheMiss")}
                              </Badge>
                              <Badge tone="queued">{parsed.algorithmProfile}</Badge>
                            </div>
                          ) : null}
                        </PanelSection>

                        <PanelSection title={t("chapters.title")} subtitle={parsed ? parsed.title : t("chapters.empty")}>
                          <div className="mb-2.5 flex flex-wrap items-center gap-1.5">
                            <Select
                              className="h-9 w-36"
                              onChange={(event) => setStatusFilter(event.target.value as LocalChapterStatus | "all")}
                              value={statusFilter}
                            >
                              {statusOptions.map((option) => (
                                <option key={option} value={option}>
                                  {option === "all" ? t("chapters.all") : t(`status.${option}`)}
                                </option>
                              ))}
                            </Select>
                            <Button onClick={() => setSelectedChapterIDs(parsed?.chapters.map((chapter) => chapter.id) ?? [])} size="sm" variant="outline">
                              {t("chapters.selectAll")}
                            </Button>
                            <Button onClick={() => setSelectedChapterIDs([])} size="sm" variant="outline">
                              {t("chapters.clear")}
                            </Button>
                            <Button
                              onClick={() =>
                                setSelectedChapterIDs(
                                  parsed?.chapters.filter((chapter) => !selectedChapterIDs.includes(chapter.id)).map((chapter) => chapter.id) ?? []
                                )
                              }
                              size="sm"
                              variant="outline"
                            >
                              {t("chapters.invert")}
                            </Button>
                          </div>

                          <div className="space-y-1.5">
                            {filteredChapters.length === 0 ? (
                              <div className="border border-dashed border-border/60 px-3 py-5 text-sm text-muted-foreground">
                                {t("chapters.empty")}
                              </div>
                            ) : (
                              filteredChapters.map((chapter) => (
                                <button
                                  className="grid w-full grid-cols-[auto,1fr,auto] items-start gap-2 border border-border/60 bg-background/58 px-3 py-2.5 text-left transition hover:border-primary/25 hover:bg-background/82"
                                  key={chapter.id}
                                  onClick={() =>
                                    setSelectedChapterIDs((current) =>
                                      current.includes(chapter.id) ? current.filter((id) => id !== chapter.id) : [...current, chapter.id]
                                    )
                                  }
                                  type="button"
                                >
                                  <Checkbox checked={selectedChapterIDs.includes(chapter.id)} onChange={() => undefined} />
                                  <div className="min-w-0">
                                    <div className="flex flex-wrap items-center gap-2">
                                      <span className="truncate text-sm font-semibold">
                                        {chapter.number} · {chapter.title}
                                      </span>
                                      <Badge tone={chapter.localStatus}>{t(`status.${chapter.localStatus}`)}</Badge>
                                    </div>
                                    <div className="mt-1 text-[11px] text-muted-foreground">
                                      {chapter.pageCount} {t("chapters.pages")} · {formatDateTime(chapter.releaseDate)}
                                    </div>
                                    {chapter.localPath ? <div className="mt-1 truncate text-[11px] text-muted-foreground">{chapter.localPath}</div> : null}
                                  </div>
                                  <span className="pt-0.5 text-[10px] font-semibold uppercase tracking-[0.16em] text-muted-foreground">
                                    {selectedChapterIDs.includes(chapter.id) ? t("chapters.ready") : t("chapters.pick")}
                                  </span>
                                </button>
                              ))
                            )}
                          </div>

                          <div className="mt-3 flex items-center gap-2">
                            <div className="text-xs text-muted-foreground">
                              {selectedChapterCount} {t("jobs.selected")}
                            </div>
                            <Button
                              className="ml-auto gap-2"
                              disabled={!canStartProcessing}
                              onClick={submitSelectedChapters}
                              size="sm"
                            >
                              <Download className="h-4 w-4" />
                              {queueMutation.isPending ? t("jobs.processing") : t("jobs.start")}
                            </Button>
                          </div>
                        </PanelSection>

                        <PanelSection title={t("jobs.title")} subtitle={t("jobs.subtitle")}>
                          <div className="mb-2.5 flex items-center justify-between gap-2">
                            <div className="text-xs text-muted-foreground">
                              {jobs.length} {t("shell.tasks")}
                            </div>
                            <Button onClick={() => void queryClient.invalidateQueries({ queryKey: ["jobs"] })} size="sm" variant="ghost">
                              <RefreshCcw className="h-4 w-4" />
                            </Button>
                          </div>
                          <div className="space-y-2">
                            {jobs.length === 0 ? (
                              <div className="border border-dashed border-border/60 px-3 py-5 text-sm text-muted-foreground">
                                {t("jobs.empty")}
                              </div>
                            ) : (
                              jobs.map((job) => {
                                const progress = progressByJob[job.jobID];
                                const ratio = progress && progress.totalPages > 0 ? (progress.downloadedPages / progress.totalPages) * 100 : 0;

                                return (
                                  <div className="border border-border/60 bg-background/58 p-3" key={job.jobID}>
                                    <div className="flex items-start justify-between gap-3">
                                      <div className="min-w-0">
                                        <div className="truncate text-sm font-semibold">{job.mangaTitle}</div>
                                        <div className="mt-1 text-[11px] text-muted-foreground">{formatDateTime(job.updatedAt)}</div>
                                      </div>
                                      <Badge tone={job.status}>{t(`jobs.status.${job.status}`)}</Badge>
                                    </div>
                                    <div className="mt-2 text-xs text-muted-foreground">
                                      {progress?.chapterTitle ?? "—"} · {progress?.currentFile ?? "—"}
                                    </div>
                                    <Progress className="mt-2.5" value={ratio} />
                                    <div className="mt-2 flex items-center justify-between text-[11px] text-muted-foreground">
                                      <span>
                                        {progress?.downloadedPages ?? 0}/{progress?.totalPages ?? 0}
                                      </span>
                                      <span>
                                        {t("jobs.speed")}: {formatBytesPerSecond(progress?.bytesPerSec ?? 0)}
                                      </span>
                                    </div>
                                    <div className="mt-3 flex flex-wrap gap-1.5">
                                      <Button onClick={() => void appAdapter.pauseJob(job.jobID)} size="sm" variant="outline">
                                        {t("jobs.pause")}
                                      </Button>
                                      <Button onClick={() => void appAdapter.resumeJob(job.jobID)} size="sm" variant="outline">
                                        {t("jobs.resume")}
                                      </Button>
                                      <Button onClick={() => void appAdapter.retryFailed(job.jobID)} size="sm" variant="outline">
                                        {t("jobs.retry")}
                                      </Button>
                                    </div>
                                    {job.lastError ? <div className="mt-2 text-[11px] text-danger">{job.lastError}</div> : null}
                                  </div>
                                );
                              })
                            )}
                          </div>
                        </PanelSection>
                      </div>
                    ) : settings ? (
                      <SettingsForm settings={settings} onSave={(nextSettings) => settingsMutation.mutate(nextSettings)} />
                    ) : (
                      <div className="p-3">
                        <div className="border border-dashed border-border/60 px-3 py-5 text-sm text-muted-foreground">
                          {t("settings.loading")}
                        </div>
                      </div>
                    )}
                  </div>
                </div>
              </section>

              <button
                className="pointer-events-auto relative w-[10px] shrink-0 cursor-col-resize"
                onMouseDown={(event) => startResize(event.clientX, paneWidth)}
                type="button"
              >
                <span className="absolute bottom-0 left-1/2 top-0 w-px -translate-x-1/2 bg-border/80 transition hover:bg-primary/60" />
              </button>
            </div>
          ) : null}
        </div>
      </div>
    </main>
  );
}

function formatLocaleBadge(settings: AppSettings | undefined) {
  if (!settings || settings.localeMode === "system") {
    return "SYS";
  }

  switch (settings.locale) {
    case "zh-CN":
      return "中";
    case "ja":
      return "日";
    default:
      return "EN";
  }
}

function RailIconButton({
  active = false,
  icon,
  label,
  onClick,
}: {
  active?: boolean;
  icon: ReactNode;
  label: string;
  onClick: () => void;
}) {
  return (
    <button
      className={cn(
        "inline-flex h-10 w-10 items-center justify-center rounded-2xl border shadow-[0_1px_0_rgba(255,255,255,0.35)_inset] transition-colors",
        active
          ? "border-foreground bg-foreground text-background"
          : "border-border/70 bg-background/80 text-muted-foreground hover:bg-muted/70 hover:text-foreground"
      )}
      onClick={onClick}
      title={label}
      type="button"
    >
      <span className="sr-only">{label}</span>
      {icon}
    </button>
  );
}

function RailUtilityButton({
  badge,
  disabled = false,
  icon,
  label,
  onClick,
  title,
}: {
  badge?: string;
  disabled?: boolean;
  icon: ReactNode;
  label: string;
  onClick: () => void;
  title?: string;
}) {
  return (
    <button
      className="relative inline-flex h-10 w-10 items-center justify-center rounded-2xl border border-border/70 bg-background/80 text-muted-foreground shadow-[0_1px_0_rgba(255,255,255,0.35)_inset] transition-colors hover:bg-muted/70 hover:text-foreground disabled:cursor-not-allowed disabled:opacity-55 disabled:hover:bg-background/80 disabled:hover:text-muted-foreground"
      disabled={disabled}
      onClick={onClick}
      title={title ?? label}
      type="button"
    >
      <span className="sr-only">{label}</span>
      {icon}
      {badge ? (
        <span className="absolute bottom-1 right-1 inline-flex h-4 min-w-[1rem] items-center justify-center rounded-full bg-foreground px-1 text-[8px] font-bold leading-none text-background">
          {badge}
        </span>
      ) : null}
    </button>
  );
}

function PanelSection({ title, subtitle, children }: { title: string; subtitle?: string; children: ReactNode }) {
  return (
    <section className="border border-border/60 bg-background/44 p-3">
      <div className="mb-3">
        <h3 className="text-xs font-black uppercase tracking-[0.18em]">{title}</h3>
        {subtitle ? <p className="mt-1 text-xs text-muted-foreground">{subtitle}</p> : null}
      </div>
      {children}
    </section>
  );
}

function LibraryGrid({
  items,
  loading,
  emptyLabel,
  onOpen,
}: {
  items: LibraryManga[];
  loading: boolean;
  emptyLabel: string;
  onOpen: (mangaID: string) => void;
}) {
  if (loading) {
    return (
      <div className="flex h-full items-center justify-center border-l border-border/40 bg-card/14 text-sm text-muted-foreground">
        Loading library…
      </div>
    );
  }

  if (items.length === 0) {
    return (
      <div className="flex h-full items-center justify-center border-l border-border/40 bg-card/14 text-sm text-muted-foreground">
        {emptyLabel}
      </div>
    );
  }

  return (
    <div className="grid h-full auto-rows-max gap-px overflow-y-auto bg-border/45 sm:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4">
      {items.map((item) => (
        <button
          className="group flex min-h-[220px] flex-col overflow-hidden bg-background/92 text-left transition hover:bg-background"
          key={item.id}
          onClick={() => onOpen(item.id)}
          type="button"
        >
          <div className="relative aspect-[4/5] overflow-hidden bg-muted">
            {item.coverImageURL ? (
              <img
                alt={item.title}
                className="h-full w-full object-cover transition duration-500 group-hover:scale-[1.02]"
                loading="lazy"
                src={item.coverImageURL}
              />
            ) : (
              <div className="flex h-full w-full items-center justify-center text-muted-foreground">
                <BookImage className="h-9 w-9" />
              </div>
            )}
            <div className="absolute inset-x-0 bottom-0 bg-gradient-to-t from-black/74 via-black/18 to-transparent px-3 py-3 text-white">
              <div className="text-[10px] font-semibold uppercase tracking-[0.18em] text-white/70">{item.chapterCount} chapters</div>
              <div className="mt-1 line-clamp-2 text-base font-black">{item.title}</div>
            </div>
          </div>
          <div className="flex flex-1 items-center justify-between gap-3 px-3 py-3">
            <div className="min-w-0">
              <div className="text-xs text-muted-foreground">{item.pageCount} pages</div>
              <div className="mt-1 truncate text-[11px] text-muted-foreground">{formatDateTime(item.lastUpdated)}</div>
            </div>
            <Telescope className="h-4.5 w-4.5 shrink-0 text-primary" />
          </div>
        </button>
      ))}
    </div>
  );
}

function SettingsForm({ settings, onSave }: { settings: AppSettings; onSave: (settings: AppSettings) => void }) {
  const { t } = useTranslation();
  const [form, setForm] = useState(settings);

  useEffect(() => {
    setForm(settings);
  }, [settings]);

  return (
    <div className="space-y-3 p-3">
      <PanelSection title={t("settings.title")} subtitle={t("settings.subtitle")}>
        <form
          className="space-y-3.5"
          onSubmit={(event) => {
            event.preventDefault();
            onSave(form);
          }}
        >
          <Field label={t("settings.outputRoot")}>
            <Input value={form.outputRoot} onChange={(event) => setForm((current) => ({ ...current, outputRoot: event.target.value }))} />
          </Field>

          <div className="grid gap-3 md:grid-cols-2">
            <Field label={t("settings.maxConcurrentDownloads")}>
              <Input
                min={1}
                onChange={(event) =>
                  setForm((current) => ({
                    ...current,
                    maxConcurrentDownloads: Number(event.target.value) || current.maxConcurrentDownloads,
                  }))
                }
                type="number"
                value={form.maxConcurrentDownloads}
              />
            </Field>

            <Field label={t("settings.retryCount")}>
              <Input
                min={0}
                onChange={(event) => setForm((current) => ({ ...current, retryCount: Number(event.target.value) || 0 }))}
                type="number"
                value={form.retryCount}
              />
            </Field>
          </div>

          <Field label={t("settings.requestTimeoutSec")}>
            <Input
              min={5}
              onChange={(event) =>
                setForm((current) => ({
                  ...current,
                  requestTimeoutSec: Number(event.target.value) || current.requestTimeoutSec,
                }))
              }
              type="number"
              value={form.requestTimeoutSec}
            />
          </Field>

          <p className="text-[11px] text-muted-foreground">{t("settings.railHint")}</p>

          <Button className="w-full" type="submit">
            {t("settings.save")}
          </Button>
        </form>
      </PanelSection>
    </div>
  );
}

function Field({ label, children }: { label: string; children: ReactNode }) {
  return (
    <label className="grid gap-1.5">
      <span className="text-[10px] font-semibold uppercase tracking-[0.18em] text-muted-foreground">{label}</span>
      {children}
    </label>
  );
}
