package download

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/sakagamijun/rawmanga-download-go/internal/contracts"
)

const LibraryAssetPrefix = "/library-files/"

type mangaManifest struct {
	relativePath string
	updatedAt    time.Time
	sourceURL    string
	reader       contracts.ReaderManifest
}

type chapterSource struct {
	id          string
	title       string
	number      float64
	sourceURL   string
	completedAt string
	localPath   string
	pages       []contracts.ReaderPage
	updatedAt   time.Time
}

func ListLibraryManga(outputRoot string) ([]contracts.LibraryManga, error) {
	entries, err := os.ReadDir(outputRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return []contracts.LibraryManga{}, nil
		}
		return nil, fmt.Errorf("read library root: %w", err)
	}

	items := make([]contracts.LibraryManga, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		mangaDir := filepath.Join(outputRoot, entry.Name())
		manifest, err := loadMangaManifest(outputRoot, mangaDir)
		if err != nil {
			return nil, err
		}
		if len(manifest.reader.Chapters) == 0 {
			continue
		}

		item := contracts.LibraryManga{
			ID:            manifest.reader.MangaID,
			Title:         manifest.reader.Title,
			SourceURL:     manifest.sourceURL,
			RelativePath:  manifest.relativePath,
			CoverImageURL: manifest.reader.CoverImageURL,
			ChapterCount:  len(manifest.reader.Chapters),
			PageCount:     manifest.reader.TotalPages,
			LastUpdated:   manifest.updatedAt.UTC().Format(time.RFC3339),
		}
		items = append(items, item)
	}

	sort.SliceStable(items, func(i, j int) bool {
		return items[i].LastUpdated > items[j].LastUpdated
	})

	return items, nil
}

func GetReaderManifest(outputRoot string, mangaID string) (contracts.ReaderManifest, error) {
	relativePath, err := decodeMangaID(mangaID)
	if err != nil {
		return contracts.ReaderManifest{}, err
	}

	mangaDir, err := resolveWithinRoot(outputRoot, relativePath)
	if err != nil {
		return contracts.ReaderManifest{}, err
	}

	manifest, err := loadMangaManifest(outputRoot, mangaDir)
	if err != nil {
		return contracts.ReaderManifest{}, err
	}

	return manifest.reader, nil
}

func ResolveLibraryAssetPath(outputRoot string, requestPath string) (string, error) {
	if !strings.HasPrefix(requestPath, LibraryAssetPrefix) {
		return "", fmt.Errorf("unsupported asset path: %s", requestPath)
	}

	relativeURLPath := strings.TrimPrefix(requestPath, LibraryAssetPrefix)
	if relativeURLPath == "" {
		return "", fmt.Errorf("empty asset path")
	}

	decodedPath, err := url.PathUnescape(relativeURLPath)
	if err != nil {
		return "", fmt.Errorf("decode asset path: %w", err)
	}

	targetPath, err := resolveWithinRoot(outputRoot, filepath.FromSlash(decodedPath))
	if err != nil {
		return "", err
	}

	if !isSupportedImagePath(targetPath) {
		return "", fmt.Errorf("unsupported asset extension: %s", targetPath)
	}

	info, err := os.Stat(targetPath)
	if err != nil {
		return "", fmt.Errorf("stat asset path: %w", err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("asset path is a directory")
	}

	return targetPath, nil
}

func AssetURLForPath(outputRoot string, filePath string) (string, error) {
	resolvedRoot, err := filepath.Abs(outputRoot)
	if err != nil {
		return "", fmt.Errorf("abs output root: %w", err)
	}
	if symlinkResolvedRoot, symlinkErr := filepath.EvalSymlinks(resolvedRoot); symlinkErr == nil {
		resolvedRoot = symlinkResolvedRoot
	}

	resolvedFilePath, err := filepath.Abs(filePath)
	if err != nil {
		return "", fmt.Errorf("abs file path: %w", err)
	}
	if symlinkResolvedFilePath, symlinkErr := filepath.EvalSymlinks(resolvedFilePath); symlinkErr == nil {
		resolvedFilePath = symlinkResolvedFilePath
	}

	relativePath, err := filepath.Rel(resolvedRoot, resolvedFilePath)
	if err != nil {
		return "", fmt.Errorf("derive asset relative path: %w", err)
	}

	relativePath = filepath.ToSlash(relativePath)
	if relativePath == "." || strings.HasPrefix(relativePath, "../") || strings.Contains(relativePath, "\x00") {
		return "", fmt.Errorf("illegal asset relative path: %s", relativePath)
	}

	segments := strings.Split(relativePath, "/")
	for index, segment := range segments {
		segments[index] = url.PathEscape(segment)
	}

	return LibraryAssetPrefix + strings.Join(segments, "/"), nil
}

func loadMangaManifest(outputRoot string, mangaDir string) (mangaManifest, error) {
	relativePath, err := filepath.Rel(outputRoot, mangaDir)
	if err != nil {
		return mangaManifest{}, fmt.Errorf("derive manga relative path: %w", err)
	}

	chapterEntries, err := os.ReadDir(mangaDir)
	if err != nil {
		return mangaManifest{}, fmt.Errorf("read manga directory: %w", err)
	}

	chapters := make([]chapterSource, 0, len(chapterEntries))
	var (
		totalPages int
		updatedAt  time.Time
		sourceURL  string
	)

	for _, chapterEntry := range chapterEntries {
		if !chapterEntry.IsDir() {
			continue
		}

		chapterDir := filepath.Join(mangaDir, chapterEntry.Name())
		source, err := loadChapterSource(outputRoot, chapterDir)
		if err != nil {
			return mangaManifest{}, err
		}
		if len(source.pages) == 0 {
			continue
		}

		if source.updatedAt.After(updatedAt) {
			updatedAt = source.updatedAt
		}
		if sourceURL == "" && source.sourceURL != "" {
			sourceURL = source.sourceURL
		}
		totalPages += len(source.pages)
		chapters = append(chapters, source)
	}

	sort.SliceStable(chapters, func(i, j int) bool {
		if chapters[i].number == chapters[j].number {
			return chapters[i].title < chapters[j].title
		}
		return chapters[i].number < chapters[j].number
	})

	readerChapters := make([]contracts.ReaderChapter, 0, len(chapters))
	startPage := 0
	coverImageURL := ""
	for _, source := range chapters {
		readerChapter := contracts.ReaderChapter{
			ID:          source.id,
			Title:       source.title,
			Number:      source.number,
			StartPage:   startPage,
			PageCount:   len(source.pages),
			Pages:       source.pages,
			LocalPath:   source.localPath,
			CompletedAt: source.completedAt,
		}
		if coverImageURL == "" && len(source.pages) > 0 {
			coverImageURL = source.pages[0].SourceURL
		}
		readerChapters = append(readerChapters, readerChapter)
		startPage += len(source.pages)
	}

	return mangaManifest{
		relativePath: filepath.ToSlash(relativePath),
		updatedAt:    updatedAt,
		sourceURL:    sourceURL,
		reader: contracts.ReaderManifest{
			MangaID:       encodeMangaID(relativePath),
			Title:         filepath.Base(mangaDir),
			CoverImageURL: coverImageURL,
			TotalPages:    totalPages,
			Chapters:      readerChapters,
		},
	}, nil
}

func loadChapterSource(outputRoot string, chapterDir string) (chapterSource, error) {
	sidecar, sidecarFound, err := ReadSidecar(SidecarPath(chapterDir))
	if err != nil {
		return chapterSource{}, err
	}

	pages := make([]contracts.ReaderPage, 0)
	if sidecarFound && len(sidecar.Files) > 0 {
		sort.SliceStable(sidecar.Files, func(i, j int) bool {
			return sidecar.Files[i].PageIndex < sidecar.Files[j].PageIndex
		})
		for _, file := range sidecar.Files {
			fullPath := filepath.Join(chapterDir, file.FileName)
			if !fileExists(fullPath) || !isSupportedImagePath(fullPath) {
				continue
			}
			sourceURL, err := AssetURLForPath(outputRoot, fullPath)
			if err != nil {
				return chapterSource{}, err
			}
			pages = append(pages, contracts.ReaderPage{
				ID:           fmt.Sprintf("%s:%03d", sidecar.ChapterID, file.PageIndex),
				ChapterID:    sidecar.ChapterID,
				ChapterTitle: sidecar.ChapterTitle,
				PageIndex:    file.PageIndex,
				FileName:     file.FileName,
				SourceURL:    sourceURL,
			})
		}
	} else {
		entries, err := os.ReadDir(chapterDir)
		if err != nil {
			return chapterSource{}, fmt.Errorf("read chapter directory: %w", err)
		}
		sort.SliceStable(entries, func(i, j int) bool {
			return entries[i].Name() < entries[j].Name()
		})
		for index, entry := range entries {
			if entry.IsDir() {
				continue
			}
			fullPath := filepath.Join(chapterDir, entry.Name())
			if !isSupportedImagePath(fullPath) {
				continue
			}
			sourceURL, err := AssetURLForPath(outputRoot, fullPath)
			if err != nil {
				return chapterSource{}, err
			}
			pages = append(pages, contracts.ReaderPage{
				ID:           fmt.Sprintf("%s:%03d", filepath.Base(chapterDir), index),
				ChapterID:    filepath.Base(chapterDir),
				ChapterTitle: filepath.Base(chapterDir),
				PageIndex:    index,
				FileName:     entry.Name(),
				SourceURL:    sourceURL,
			})
		}
	}

	number := inferChapterNumber(filepath.Base(chapterDir))
	title := filepath.Base(chapterDir)
	chapterID := filepath.Base(chapterDir)
	completedAt := ""
	if sidecarFound {
		number = sidecar.ChapterNumber
		if sidecar.ChapterTitle != "" {
			title = sidecar.ChapterTitle
		}
		if sidecar.ChapterID != "" {
			chapterID = sidecar.ChapterID
		}
		completedAt = sidecar.CompletedAt
	}

	info, err := os.Stat(chapterDir)
	if err != nil {
		return chapterSource{}, fmt.Errorf("stat chapter directory: %w", err)
	}

	return chapterSource{
		id:          chapterID,
		title:       title,
		number:      number,
		sourceURL:   sidecar.SourceURL,
		completedAt: completedAt,
		localPath:   chapterDir,
		pages:       pages,
		updatedAt:   info.ModTime(),
	}, nil
}

func resolveWithinRoot(root string, relativePath string) (string, error) {
	if relativePath == "" {
		return "", fmt.Errorf("empty relative path")
	}

	cleanedPath := filepath.Clean(relativePath)
	if cleanedPath == "." || filepath.IsAbs(cleanedPath) || strings.Contains(cleanedPath, "\x00") {
		return "", fmt.Errorf("illegal relative path: %s", relativePath)
	}

	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("abs root: %w", err)
	}
	absoluteTarget := filepath.Join(absoluteRoot, cleanedPath)
	absoluteTarget, err = filepath.Abs(absoluteTarget)
	if err != nil {
		return "", fmt.Errorf("abs target: %w", err)
	}

	resolvedRoot, err := filepath.EvalSymlinks(absoluteRoot)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("eval root symlinks: %w", err)
	}
	if resolvedRoot == "" {
		resolvedRoot = absoluteRoot
	}

	resolvedTarget, err := filepath.EvalSymlinks(absoluteTarget)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("eval target symlinks: %w", err)
	}
	if resolvedTarget == "" {
		resolvedTarget = absoluteTarget
	}

	relativeToRoot, err := filepath.Rel(resolvedRoot, resolvedTarget)
	if err != nil {
		return "", fmt.Errorf("derive root-relative path: %w", err)
	}
	if relativeToRoot == "." || strings.HasPrefix(relativeToRoot, ".."+string(filepath.Separator)) || relativeToRoot == ".." {
		return "", fmt.Errorf("path escapes manga root")
	}

	return resolvedTarget, nil
}

func encodeMangaID(relativePath string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(filepath.ToSlash(relativePath)))
}

func decodeMangaID(identifier string) (string, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(identifier)
	if err != nil {
		return "", fmt.Errorf("decode manga id: %w", err)
	}
	return filepath.FromSlash(string(decoded)), nil
}

func inferChapterNumber(chapterDirName string) float64 {
	label := chapterDirName
	if dashIndex := strings.Index(label, " - "); dashIndex >= 0 {
		label = label[:dashIndex]
	}
	label = strings.ReplaceAll(label, "_", ".")
	value, err := strconv.ParseFloat(label, 64)
	if err != nil {
		return 0
	}
	return value
}

func isSupportedImagePath(filePath string) bool {
	switch strings.ToLower(path.Ext(filepath.ToSlash(filePath))) {
	case ".jpg", ".jpeg", ".png", ".webp", ".gif", ".avif":
		return true
	default:
		return false
	}
}

func fileExists(filePath string) bool {
	info, err := os.Stat(filePath)
	return err == nil && !info.IsDir()
}
