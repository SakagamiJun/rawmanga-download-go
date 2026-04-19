package klz9

import (
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/sakagamijun/rawmanga-download-go/internal/contracts"
)

func TestExtractBundleFromHTML(t *testing.T) {
	htmlContent, err := os.ReadFile(filepath.Join("..", "..", "otona-ni-narenai-bokura-wa.html"))
	if err != nil {
		t.Fatalf("read html fixture: %v", err)
	}

	pageURL, err := url.Parse("https://klz9.com/otona-ni-narenai-bokura-wa.html")
	if err != nil {
		t.Fatalf("parse page url: %v", err)
	}

	bundleURL, bundleHash, err := extractBundleFromHTML(pageURL, string(htmlContent))
	if err != nil {
		t.Fatalf("extract bundle: %v", err)
	}

	if bundleURL != "https://klz9.com/assets/index-BBvPdTHw.js.pagespeed.ce.WKBIIa11t7.js" {
		t.Fatalf("unexpected bundle url: %s", bundleURL)
	}

	if bundleHash != "index-BBvPdTHw.js.pagespeed.ce.WKBIIa11t7.js" {
		t.Fatalf("unexpected bundle hash: %s", bundleHash)
	}
}

func TestExtractSiteProfile(t *testing.T) {
	bundleContent, err := os.ReadFile(filepath.Join("..", "..", "assets", "index-BBvPdTHw.js.pagespeed.ce.WKBIIa11t7.js"))
	if err != nil {
		t.Fatalf("read bundle fixture: %v", err)
	}

	profile, err := extractSiteProfile(
		"https://klz9.com/assets/index-BBvPdTHw.js.pagespeed.ce.WKBIIa11t7.js",
		"index-BBvPdTHw.js.pagespeed.ce.WKBIIa11t7.js",
		string(bundleContent),
	)
	if err != nil {
		t.Fatalf("extract site profile: %v", err)
	}

	if profile.APIBase != "https://klz9.com/api" {
		t.Fatalf("unexpected api base: %s", profile.APIBase)
	}

	if profile.SignatureSecret != "KL9K40zaSyC9K40vOMLLbEcepIFBhUKXwELqxlwTEF" {
		t.Fatalf("unexpected signature secret: %s", profile.SignatureSecret)
	}

	if profile.ImageHostRewrite["https://imfaclub.com"] != "https://j1.jfimv2.xyz" {
		t.Fatalf("unexpected final host rewrite for imfaclub.com: %#v", profile.ImageHostRewrite)
	}

	if profile.ImageHostRewrite["https://s4.ihlv1.xyz"] != "https://j4.jfimv2.xyz" {
		t.Fatalf("unexpected final host rewrite for s4.ihlv1.xyz: %#v", profile.ImageHostRewrite)
	}

	if len(profile.IgnorePageURLs) != 3 {
		t.Fatalf("unexpected ignore list length: %d", len(profile.IgnorePageURLs))
	}
}

func TestCleanPagesAppliesFinalHostRewriteAndIgnoreRules(t *testing.T) {
	profile := contractsFixture()
	pages := cleanPages([]string{
		"https://imfaclub.com/images/1.jpg",
		"https://s4.ihlv1.xyz/images/2.jpg",
		"https://1.bp.blogspot.com/-ZMyVQcnjYyE/W2cRdXQb15I/AAAAAAACDnk/8X1Hm7wmhz4hLvpIzTNBHQnhuKu05Qb0gCHMYCw/s0/LHScan.png",
	}, profile)

	if len(pages) != 2 {
		t.Fatalf("unexpected page count after cleaning: %d", len(pages))
	}

	if pages[0] != "https://j1.jfimv2.xyz/images/1.jpg" {
		t.Fatalf("unexpected rewritten page 1: %s", pages[0])
	}

	if pages[1] != "https://j4.jfimv2.xyz/images/2.jpg" {
		t.Fatalf("unexpected rewritten page 2: %s", pages[1])
	}
}

func contractsFixture() contracts.SiteProfile {
	return contracts.SiteProfile{
		IgnorePageURLs: []string{
			"https://1.bp.blogspot.com/-ZMyVQcnjYyE/W2cRdXQb15I/AAAAAAACDnk/8X1Hm7wmhz4hLvpIzTNBHQnhuKu05Qb0gCHMYCw/s0/LHScan.png",
		},
		ImageHostRewrite: map[string]string{
			"https://imfaclub.com": "https://j1.jfimv2.xyz",
			"https://s4.ihlv1.xyz": "https://j4.jfimv2.xyz",
		},
	}
}
