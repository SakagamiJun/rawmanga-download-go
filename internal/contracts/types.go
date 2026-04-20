package contracts

type LocaleMode string

const (
	LocaleModeSystem LocaleMode = "system"
	LocaleModeManual LocaleMode = "manual"
)

type ThemeMode string

const (
	ThemeModeSystem ThemeMode = "system"
	ThemeModeLight  ThemeMode = "light"
	ThemeModeDark   ThemeMode = "dark"
)

type LocalChapterStatus string

const (
	LocalChapterStatusNotDownloaded LocalChapterStatus = "not_downloaded"
	LocalChapterStatusPartial       LocalChapterStatus = "partial"
	LocalChapterStatusComplete      LocalChapterStatus = "complete"
	LocalChapterStatusMissing       LocalChapterStatus = "missing"
)

type JobStatus string

const (
	JobStatusQueued    JobStatus = "queued"
	JobStatusRunning   JobStatus = "running"
	JobStatusPaused    JobStatus = "paused"
	JobStatusCompleted JobStatus = "completed"
	JobStatusFailed    JobStatus = "failed"
	JobStatusCanceled  JobStatus = "canceled"
	JobStatusSkipped   JobStatus = "skipped"
)

type LocalSummary struct {
	NotDownloaded int `json:"notDownloaded"`
	Partial       int `json:"partial"`
	Complete      int `json:"complete"`
	Missing       int `json:"missing"`
}

type SiteProfile struct {
	Site             string            `json:"site"`
	BundleURL        string            `json:"bundleURL"`
	BundleHash       string            `json:"bundleHash"`
	APIBase          string            `json:"apiBase"`
	SignatureSecret  string            `json:"signatureSecret"`
	SignatureMode    string            `json:"signatureMode"`
	ImageHostRewrite map[string]string `json:"imageHostRewrite"`
	IgnorePageURLs   []string          `json:"ignorePageURLs"`
	ExtractedAt      string            `json:"extractedAt"`
}

type ChapterItem struct {
	ID          string             `json:"id"`
	Number      float64            `json:"number"`
	Title       string             `json:"title"`
	ReleaseDate string             `json:"releaseDate"`
	PageCount   int                `json:"pageCount"`
	LocalStatus LocalChapterStatus `json:"localStatus"`
	LocalPath   string             `json:"localPath"`
	Selected    bool               `json:"selected"`
}

type ParsedMangaResult struct {
	SourceURL        string        `json:"sourceURL"`
	Slug             string        `json:"slug"`
	Title            string        `json:"title"`
	CoverURL         string        `json:"coverURL"`
	Chapters         []ChapterItem `json:"chapters"`
	LocalSummary     LocalSummary  `json:"localSummary"`
	ProfileCacheHit  bool          `json:"profileCacheHit"`
	AlgorithmProfile string        `json:"algorithmProfile"`
}

type QueueDownloadRequest struct {
	SourceURL  string   `json:"sourceURL"`
	MangaSlug  string   `json:"mangaSlug"`
	Title      string   `json:"title"`
	ChapterIDs []string `json:"chapterIDs"`
	OutputRoot string   `json:"outputRoot"`
}

type DownloadJob struct {
	JobID              string    `json:"jobID"`
	MangaSlug          string    `json:"mangaSlug"`
	MangaTitle         string    `json:"mangaTitle"`
	SourceURL          string    `json:"sourceURL"`
	Status             JobStatus `json:"status"`
	QueuedChapters     int       `json:"queuedChapters"`
	CompletedChapters  int       `json:"completedChapters"`
	FailedChapters     int       `json:"failedChapters"`
	CreatedAt          string    `json:"createdAt"`
	UpdatedAt          string    `json:"updatedAt"`
	LastError          string    `json:"lastError"`
	MaxConcurrentPages int       `json:"maxConcurrentPages"`
}

type DownloadProgressEvent struct {
	JobID           string    `json:"jobID"`
	ChapterID       string    `json:"chapterID"`
	ChapterTitle    string    `json:"chapterTitle"`
	PageIndex       int       `json:"pageIndex"`
	Status          JobStatus `json:"status"`
	DownloadedPages int       `json:"downloadedPages"`
	TotalPages      int       `json:"totalPages"`
	CurrentFile     string    `json:"currentFile"`
	BytesPerSec     int64     `json:"bytesPerSec"`
	Error           string    `json:"error"`
	LocalStatus     string    `json:"localStatus"`
	CompletedAt     string    `json:"completedAt"`
}

type LocalChapterState struct {
	ChapterID         string             `json:"chapterID"`
	ChapterNumber     float64            `json:"chapterNumber"`
	Title             string             `json:"title"`
	Status            LocalChapterStatus `json:"status"`
	LocalPath         string             `json:"localPath"`
	LocalPageCount    int                `json:"localPageCount"`
	ExpectedPageCount int                `json:"expectedPageCount"`
}

type AppSettings struct {
	OutputRoot                string     `json:"outputRoot"`
	MaxConcurrentDownloads    int        `json:"maxConcurrentDownloads"`
	RetryCount                int        `json:"retryCount"`
	RequestTimeoutSec         int        `json:"requestTimeoutSec"`
	LocaleMode                LocaleMode `json:"localeMode"`
	Locale                    string     `json:"locale"`
	ThemeMode                 ThemeMode  `json:"themeMode"`
	ReaderScrollCachePages    int        `json:"readerScrollCachePages"`
	AutoRestoreReaderProgress bool       `json:"autoRestoreReaderProgress"`
}

type LibraryManga struct {
	ID            string `json:"id"`
	Title         string `json:"title"`
	RelativePath  string `json:"relativePath"`
	CoverImageURL string `json:"coverImageURL"`
	ChapterCount  int    `json:"chapterCount"`
	PageCount     int    `json:"pageCount"`
	LastUpdated   string `json:"lastUpdated"`
}

type ReaderManifest struct {
	MangaID       string          `json:"mangaID"`
	Title         string          `json:"title"`
	CoverImageURL string          `json:"coverImageURL"`
	TotalPages    int             `json:"totalPages"`
	Chapters      []ReaderChapter `json:"chapters"`
}

type ReaderChapter struct {
	ID          string       `json:"id"`
	Title       string       `json:"title"`
	Number      float64      `json:"number"`
	StartPage   int          `json:"startPage"`
	PageCount   int          `json:"pageCount"`
	Pages       []ReaderPage `json:"pages"`
	LocalPath   string       `json:"localPath"`
	CompletedAt string       `json:"completedAt"`
}

type ReaderPage struct {
	ID           string `json:"id"`
	ChapterID    string `json:"chapterID"`
	ChapterTitle string `json:"chapterTitle"`
	PageIndex    int    `json:"pageIndex"`
	FileName     string `json:"fileName"`
	SourceURL    string `json:"sourceURL"`
}

type ReaderProgress struct {
	MangaID   string `json:"mangaID"`
	ChapterID string `json:"chapterID"`
	Page      int    `json:"page"`
	UpdatedAt string `json:"updatedAt"`
}
