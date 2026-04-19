package download

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

type ChapterSidecar struct {
	SourceURL         string               `json:"sourceURL"`
	MangaSlug         string               `json:"mangaSlug"`
	MangaTitle        string               `json:"mangaTitle"`
	ChapterID         string               `json:"chapterID"`
	ChapterNumber     float64              `json:"chapterNumber"`
	ChapterTitle      string               `json:"chapterTitle"`
	BundleHash        string               `json:"bundleHash"`
	ExpectedPageCount int                  `json:"expectedPageCount"`
	DownloadedPages   int                  `json:"downloadedPages"`
	Files             []ChapterSidecarFile `json:"files"`
	CompletedAt       string               `json:"completedAt"`
}

type ChapterSidecarFile struct {
	PageIndex int    `json:"pageIndex"`
	FileName  string `json:"fileName"`
	URL       string `json:"url"`
}

var invalidPathChars = regexp.MustCompile(`[<>:"/\\|?*\x00-\x1F]+`)

func MangaDirectory(root string, mangaTitle string) string {
	return filepath.Join(root, sanitizePathPart(mangaTitle))
}

func ChapterDirectory(root string, number float64, title string) string {
	label := formatChapterNumber(number)
	return filepath.Join(root, fmt.Sprintf("%s - %s", label, sanitizePathPart(title)))
}

func SidecarPath(chapterDir string) string {
	return filepath.Join(chapterDir, ".klz9-chapter.json")
}

func ImageFilename(pageIndex int, ext string) string {
	if ext == "" {
		ext = ".jpg"
	}

	return fmt.Sprintf("%03d%s", pageIndex+1, ext)
}

func WriteSidecar(path string, sidecar ChapterSidecar) error {
	payload, err := json.MarshalIndent(sidecar, "", "  ")
	if err != nil {
		return fmt.Errorf("encode sidecar: %w", err)
	}

	if err := os.WriteFile(path, payload, 0o644); err != nil {
		return fmt.Errorf("write sidecar: %w", err)
	}

	return nil
}

func ReadSidecar(path string) (ChapterSidecar, bool, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return ChapterSidecar{}, false, nil
		}

		return ChapterSidecar{}, false, fmt.Errorf("read sidecar: %w", err)
	}

	var sidecar ChapterSidecar
	if err := json.Unmarshal(payload, &sidecar); err != nil {
		return ChapterSidecar{}, false, fmt.Errorf("decode sidecar: %w", err)
	}

	return sidecar, true, nil
}

func sanitizePathPart(value string) string {
	value = strings.TrimSpace(value)
	value = invalidPathChars.ReplaceAllString(value, " ")
	value = strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return ' '
		}
		return r
	}, value)
	value = strings.Join(strings.Fields(value), " ")
	if value == "" {
		return "untitled"
	}

	return value
}

func formatChapterNumber(number float64) string {
	if number == float64(int64(number)) {
		return fmt.Sprintf("%03d", int(number))
	}

	text := strconv.FormatFloat(number, 'f', -1, 64)
	text = strings.ReplaceAll(text, ".", "_")
	return text
}
