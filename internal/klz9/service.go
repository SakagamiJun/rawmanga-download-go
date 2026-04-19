package klz9

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/sakagamijun/rawmanga-download-go/internal/contracts"
	"github.com/sakagamijun/rawmanga-download-go/internal/store"
	"golang.org/x/net/html"
)

const (
	siteName  = "klz9"
	userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) KLZ9Downloader/0.1 Safari/537.36"
)

var (
	apiBaseRe         = regexp.MustCompile(`apiUrl:"([^"]+)"`)
	secretRe          = regexp.MustCompile(`const rr="([^"]+)"`)
	ignoreListRe      = regexp.MustCompile(`a=a\.filter\(i=>!\[(.*?)\]\.includes\(i\)\)\.map`)
	rewriteSegmentRe  = regexp.MustCompile(`\.replace\("([^"]+)","([^"]+)"\)`)
	htmlTagRe         = regexp.MustCompile(`<[^>]+>`)
	bundleScriptStart = regexp.MustCompile(`/assets/index-[^"']+\.js`)
)

type Service struct {
	store      *store.SQLiteStore
	client     *http.Client
	profileDir string
}

type ResolvedManga struct {
	SourceURL       string
	Slug            string
	Title           string
	CoverURL        string
	Chapters        []ResolvedChapter
	Profile         contracts.SiteProfile
	ProfileCacheHit bool
}

type ResolvedChapter struct {
	ID          string
	Number      float64
	Title       string
	ReleaseDate string
	Pages       []string
}

func NewService(store *store.SQLiteStore, timeoutSeconds int) (*Service, error) {
	if timeoutSeconds <= 0 {
		timeoutSeconds = 30
	}

	profileDir := filepath.Join(store.DataDir(), "profiles", siteName)
	if err := os.MkdirAll(profileDir, 0o755); err != nil {
		return nil, fmt.Errorf("create profile dir: %w", err)
	}

	return &Service{
		store:      store,
		client:     &http.Client{Timeout: time.Duration(timeoutSeconds) * time.Second},
		profileDir: profileDir,
	}, nil
}

func (s *Service) SetTimeout(timeoutSeconds int) {
	if timeoutSeconds <= 0 {
		timeoutSeconds = 30
	}

	s.client.Timeout = time.Duration(timeoutSeconds) * time.Second
}

func (s *Service) ResolveManga(ctx context.Context, inputURL string) (ResolvedManga, error) {
	parsedURL, err := normalizeInputURL(inputURL)
	if err != nil {
		return ResolvedManga{}, err
	}

	htmlContent, err := s.fetchText(ctx, parsedURL.String())
	if err != nil {
		return ResolvedManga{}, err
	}

	profile, cacheHit, err := s.resolveSiteProfile(ctx, parsedURL, htmlContent)
	if err != nil {
		return ResolvedManga{}, err
	}

	slug := mangaSlugFromPath(parsedURL.Path)
	if slug == "" {
		return ResolvedManga{}, contracts.ContractError{
			Code:    contracts.ErrCodeInvalidURL,
			Message: "unable to infer manga slug from url",
		}
	}

	mangaResponse, err := s.fetchJSON(ctx, profile, "/manga/slug/"+slug)
	if err != nil {
		return ResolvedManga{}, err
	}

	normalized := normalizeManga(parsedURL.String(), slug, mangaResponse, profile)
	normalized.Profile = profile
	normalized.ProfileCacheHit = cacheHit
	return normalized, nil
}

func (s *Service) FetchChapterByID(ctx context.Context, profile contracts.SiteProfile, chapterID string) (ResolvedChapter, error) {
	chapterResponse, err := s.fetchJSON(ctx, profile, "/chapter/"+chapterID)
	if err != nil {
		return ResolvedChapter{}, err
	}

	return normalizeChapter(chapterResponse, profile), nil
}

func (s *Service) resolveSiteProfile(ctx context.Context, pageURL *url.URL, htmlContent string) (contracts.SiteProfile, bool, error) {
	bundleURL, bundleHash, err := extractBundleFromHTML(pageURL, htmlContent)
	if err != nil {
		return contracts.SiteProfile{}, false, err
	}

	if profile, found, err := s.store.GetSiteProfile(bundleHash); err == nil && found {
		return profile, true, nil
	} else if err != nil {
		return contracts.SiteProfile{}, false, err
	}

	if profile, found, err := s.readProfileFromDisk(bundleHash); err == nil && found {
		if err := s.store.SaveSiteProfile(profile); err != nil {
			return contracts.SiteProfile{}, false, err
		}

		return profile, true, nil
	} else if err != nil {
		return contracts.SiteProfile{}, false, err
	}

	bundleContent, err := s.fetchText(ctx, bundleURL)
	if err != nil {
		return contracts.SiteProfile{}, false, err
	}

	profile, err := extractSiteProfile(bundleURL, bundleHash, bundleContent)
	if err != nil {
		return contracts.SiteProfile{}, false, err
	}

	if err := s.store.SaveSiteProfile(profile); err != nil {
		return contracts.SiteProfile{}, false, err
	}

	if err := s.writeProfileToDisk(profile); err != nil {
		return contracts.SiteProfile{}, false, err
	}

	return profile, false, nil
}

func (s *Service) fetchText(ctx context.Context, rawURL string) (string, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	request.Header.Set("User-Agent", userAgent)
	request.Header.Set("Accept", "text/html,application/javascript,application/json;q=0.9,*/*;q=0.8")

	response, err := s.client.Do(request)
	if err != nil {
		return "", fmt.Errorf("request %s: %w", rawURL, err)
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", fmt.Errorf("request %s: unexpected status %d", rawURL, response.StatusCode)
	}

	payload, err := io.ReadAll(response.Body)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", rawURL, err)
	}

	return string(payload), nil
}

func (s *Service) fetchJSON(ctx context.Context, profile contracts.SiteProfile, endpoint string) (map[string]any, error) {
	if profile.APIBase == "" || profile.SignatureSecret == "" {
		return nil, contracts.ContractError{
			Code:    contracts.ErrCodeProfileNotFound,
			Message: "site profile is missing api base or signature secret",
		}
	}

	requestURL := strings.TrimRight(profile.APIBase, "/") + endpoint
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create signed request: %w", err)
	}

	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("User-Agent", userAgent)
	request.Header.Set("x-client-ts", timestamp)
	request.Header.Set("x-client-sig", signTimestamp(timestamp, profile.SignatureSecret))

	response, err := s.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("request %s: %w", requestURL, err)
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusNotFound {
		return nil, contracts.ContractError{
			Code:    contracts.ErrCodeMangaNotFound,
			Message: fmt.Sprintf("resource not found: %s", endpoint),
		}
	}

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("request %s: unexpected status %d", requestURL, response.StatusCode)
	}

	var payload map[string]any
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode %s: %w", requestURL, err)
	}

	return payload, nil
}

func signTimestamp(timestamp string, secret string) string {
	sum := sha256.Sum256([]byte(timestamp + "." + secret))
	return hex.EncodeToString(sum[:])
}

func extractBundleFromHTML(pageURL *url.URL, htmlContent string) (string, string, error) {
	root, err := html.Parse(strings.NewReader(htmlContent))
	if err == nil {
		var walk func(*html.Node) string
		walk = func(node *html.Node) string {
			if node.Type == html.ElementNode && node.Data == "script" {
				for _, attribute := range node.Attr {
					if attribute.Key == "src" && strings.Contains(attribute.Val, "/assets/index-") && strings.Contains(attribute.Val, ".js") {
						return attribute.Val
					}
				}
			}

			for child := node.FirstChild; child != nil; child = child.NextSibling {
				if match := walk(child); match != "" {
					return match
				}
			}

			return ""
		}

		if match := walk(root); match != "" {
			resolved := pageURL.ResolveReference(&url.URL{Path: match})
			return resolved.String(), path.Base(match), nil
		}
	}

	match := bundleScriptStart.FindString(htmlContent)
	if match == "" {
		return "", "", contracts.ContractError{
			Code:    contracts.ErrCodeProfileNotFound,
			Message: "unable to find main bundle in html",
		}
	}

	resolved := pageURL.ResolveReference(&url.URL{Path: match})
	return resolved.String(), path.Base(match), nil
}

func extractSiteProfile(bundleURL string, bundleHash string, bundleContent string) (contracts.SiteProfile, error) {
	apiBase := findSubmatch(apiBaseRe, bundleContent)
	secret := findSubmatch(secretRe, bundleContent)
	if apiBase == "" || secret == "" {
		return contracts.SiteProfile{}, contracts.ContractError{
			Code:    contracts.ErrCodeProfileNotFound,
			Message: "unable to extract api base or signature secret from bundle",
		}
	}

	rewriteSegment := bundleContent
	if start := strings.Index(bundleContent, `Array.isArray(t.pages)?a=t.pages`); start >= 0 {
		if end := strings.Index(bundleContent[start:], `const s=t.last_update`); end >= 0 {
			rewriteSegment = bundleContent[start : start+end]
		}
	}

	pairs := rewriteSegmentRe.FindAllStringSubmatch(rewriteSegment, -1)
	rewriteMap := toFinalRewriteMap(pairs)
	ignorePageURLs := extractIgnoreURLs(bundleContent)

	return contracts.SiteProfile{
		Site:             siteName,
		BundleURL:        bundleURL,
		BundleHash:       bundleHash,
		APIBase:          apiBase,
		SignatureSecret:  secret,
		SignatureMode:    "sha256(ts + \".\" + secret)",
		ImageHostRewrite: rewriteMap,
		IgnorePageURLs:   ignorePageURLs,
		ExtractedAt:      time.Now().UTC().Format(time.RFC3339),
	}, nil
}

func extractIgnoreURLs(bundleContent string) []string {
	match := ignoreListRe.FindStringSubmatch(bundleContent)
	if len(match) < 2 {
		return nil
	}

	var ignore []string
	for _, item := range strings.Split(match[1], ",") {
		trimmed := strings.Trim(item, `"' `)
		if trimmed != "" {
			ignore = append(ignore, trimmed)
		}
	}

	return ignore
}

func toFinalRewriteMap(pairs [][]string) map[string]string {
	raw := make(map[string]string)
	order := make([]string, 0, len(pairs))

	for _, pair := range pairs {
		if len(pair) < 3 {
			continue
		}

		raw[pair[1]] = pair[2]
		order = append(order, pair[1])
	}

	resolve := func(source string) string {
		seen := map[string]struct{}{}
		current := source
		target, ok := raw[current]
		for ok {
			if _, exists := seen[current]; exists {
				return target
			}

			seen[current] = struct{}{}
			next, nextExists := raw[target]
			if !nextExists {
				return target
			}

			current = target
			target = next
			ok = true
		}

		return raw[source]
	}

	finalMap := make(map[string]string, len(raw))
	for _, source := range order {
		finalMap[source] = resolve(source)
	}

	return finalMap
}

func (s *Service) readProfileFromDisk(bundleHash string) (contracts.SiteProfile, bool, error) {
	filePath := filepath.Join(s.profileDir, bundleHash+".json")
	payload, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return contracts.SiteProfile{}, false, nil
		}

		return contracts.SiteProfile{}, false, fmt.Errorf("read profile from disk: %w", err)
	}

	var profile contracts.SiteProfile
	if err := json.Unmarshal(payload, &profile); err != nil {
		return contracts.SiteProfile{}, false, fmt.Errorf("decode profile from disk: %w", err)
	}

	return profile, true, nil
}

func (s *Service) writeProfileToDisk(profile contracts.SiteProfile) error {
	filePath := filepath.Join(s.profileDir, profile.BundleHash+".json")
	payload, err := json.MarshalIndent(profile, "", "  ")
	if err != nil {
		return fmt.Errorf("encode profile to disk: %w", err)
	}

	if err := os.WriteFile(filePath, payload, 0o644); err != nil {
		return fmt.Errorf("write profile to disk: %w", err)
	}

	return nil
}

func normalizeManga(sourceURL string, slug string, payload map[string]any, profile contracts.SiteProfile) ResolvedManga {
	title := firstNonEmpty(getString(payload, "name"), getString(payload, "title"), slug)
	coverURL := firstNonEmpty(getString(payload, "cover"), getString(payload, "thumbnail"))

	chaptersValue, _ := payload["chapters"].([]any)
	chapters := make([]ResolvedChapter, 0, len(chaptersValue))
	for _, rawChapter := range chaptersValue {
		chapterMap, ok := rawChapter.(map[string]any)
		if !ok {
			continue
		}

		chapters = append(chapters, normalizeChapter(chapterMap, profile))
	}

	sort.SliceStable(chapters, func(i, j int) bool {
		return chapters[i].Number > chapters[j].Number
	})

	return ResolvedManga{
		SourceURL: sourceURL,
		Slug:      slug,
		Title:     title,
		CoverURL:  coverURL,
		Chapters:  chapters,
	}
}

func normalizeChapter(payload map[string]any, profile contracts.SiteProfile) ResolvedChapter {
	number := getFloat(payload, "chapter")
	pages := extractPages(payload, profile)

	return ResolvedChapter{
		ID:          firstNonEmpty(getString(payload, "id")),
		Number:      number,
		Title:       firstNonEmpty(getString(payload, "name"), fmt.Sprintf("Chapter %.0f", number)),
		ReleaseDate: normalizeTimestamp(firstNonEmpty(getString(payload, "last_update"), getString(payload, "updated_at"))),
		Pages:       pages,
	}
}

func extractPages(payload map[string]any, profile contracts.SiteProfile) []string {
	if pages, ok := payload["pages"]; ok {
		if extracted := toStringSlice(pages); len(extracted) > 0 {
			return cleanPages(extracted, profile)
		}
	}

	content := strings.TrimSpace(getString(payload, "content"))
	if content == "" {
		return nil
	}

	if strings.HasPrefix(content, "[") {
		var items []string
		if err := json.Unmarshal([]byte(content), &items); err == nil {
			return cleanPages(items, profile)
		}
	}

	if strings.HasPrefix(content, "{") {
		var objectPayload map[string]any
		if err := json.Unmarshal([]byte(content), &objectPayload); err == nil {
			if extracted := toStringSlice(objectPayload["pages"]); len(extracted) > 0 {
				return cleanPages(extracted, profile)
			}
		}
	}

	var items []string
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "http://") || strings.HasPrefix(trimmed, "https://") {
			items = append(items, trimmed)
		}
	}

	return cleanPages(items, profile)
}

func cleanPages(items []string, profile contracts.SiteProfile) []string {
	if len(items) == 0 {
		return nil
	}

	ignoreSet := make(map[string]struct{}, len(profile.IgnorePageURLs))
	for _, item := range profile.IgnorePageURLs {
		ignoreSet[item] = struct{}{}
	}

	keys := make([]string, 0, len(profile.ImageHostRewrite))
	for key := range profile.ImageHostRewrite {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		return len(keys[i]) > len(keys[j])
	})

	var clean []string
	for _, item := range items {
		normalized := strings.Replace(item, "http://", "https://", 1)
		if _, ignored := ignoreSet[normalized]; ignored {
			continue
		}
		if _, ignored := ignoreSet[item]; ignored {
			continue
		}

		for _, key := range keys {
			if strings.Contains(normalized, key) {
				normalized = strings.Replace(normalized, key, profile.ImageHostRewrite[key], 1)
			}
		}

		clean = append(clean, normalized)
	}

	return clean
}

func normalizeInputURL(input string) (*url.URL, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, contracts.ContractError{
			Code:    contracts.ErrCodeInvalidURL,
			Message: "url is required",
		}
	}

	parsedURL, err := url.Parse(input)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return nil, contracts.ContractError{
			Code:    contracts.ErrCodeInvalidURL,
			Message: "input must be a valid absolute URL",
		}
	}

	return parsedURL, nil
}

func mangaSlugFromPath(rawPath string) string {
	base := path.Base(rawPath)
	base = strings.TrimSuffix(base, path.Ext(base))
	base = strings.TrimSpace(base)
	if base == "" || strings.Contains(base, "-chapter-") {
		return ""
	}

	return base
}

func normalizeTimestamp(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	candidates := []string{
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, candidate := range candidates {
		if parsed, err := time.Parse(candidate, raw); err == nil {
			return parsed.UTC().Format(time.RFC3339)
		}
	}

	return raw
}

func firstNonEmpty(items ...string) string {
	for _, item := range items {
		if strings.TrimSpace(item) != "" {
			return strings.TrimSpace(item)
		}
	}

	return ""
}

func getString(payload map[string]any, key string) string {
	value, ok := payload[key]
	if !ok || value == nil {
		return ""
	}

	switch typed := value.(type) {
	case string:
		if key == "description" {
			return strings.TrimSpace(htmlTagRe.ReplaceAllString(typed, ""))
		}
		return strings.TrimSpace(typed)
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	case int:
		return strconv.Itoa(typed)
	default:
		return fmt.Sprintf("%v", typed)
	}
}

func getFloat(payload map[string]any, key string) float64 {
	value, ok := payload[key]
	if !ok || value == nil {
		return 0
	}

	switch typed := value.(type) {
	case float64:
		return typed
	case int:
		return float64(typed)
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		if err == nil {
			return parsed
		}
	}

	return 0
}

func toStringSlice(value any) []string {
	items, ok := value.([]any)
	if !ok {
		return nil
	}

	result := make([]string, 0, len(items))
	for _, item := range items {
		text, ok := item.(string)
		if ok && strings.TrimSpace(text) != "" {
			result = append(result, strings.TrimSpace(text))
		}
	}

	return result
}

func findSubmatch(expression *regexp.Regexp, input string) string {
	match := expression.FindStringSubmatch(input)
	if len(match) < 2 {
		return ""
	}

	return strings.TrimSpace(match[1])
}
