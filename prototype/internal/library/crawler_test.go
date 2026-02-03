package library

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestParseSitemap_URLSet(t *testing.T) {
	sitemap := `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url>
    <loc>https://example.com/page1</loc>
  </url>
  <url>
    <loc>https://example.com/page2</loc>
  </url>
  <url>
    <loc>https://example.com/page3</loc>
  </url>
</urlset>`

	urls, err := parseSitemap([]byte(sitemap))
	if err != nil {
		t.Fatalf("parseSitemap failed: %v", err)
	}

	if len(urls) != 3 {
		t.Errorf("expected 3 URLs, got %d", len(urls))
	}

	expected := []string{
		"https://example.com/page1",
		"https://example.com/page2",
		"https://example.com/page3",
	}

	for i, want := range expected {
		if i >= len(urls) {
			break
		}
		if urls[i] != want {
			t.Errorf("URL[%d] = %q, want %q", i, urls[i], want)
		}
	}
}

func TestParseSitemap_Index(t *testing.T) {
	sitemap := `<?xml version="1.0" encoding="UTF-8"?>
<sitemapindex xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <sitemap>
    <loc>https://example.com/sitemap1.xml</loc>
  </sitemap>
  <sitemap>
    <loc>https://example.com/sitemap2.xml</loc>
  </sitemap>
</sitemapindex>`

	urls, err := parseSitemap([]byte(sitemap))
	if err != nil {
		t.Fatalf("parseSitemap failed: %v", err)
	}

	if len(urls) != 2 {
		t.Errorf("expected 2 URLs, got %d", len(urls))
	}
}

func TestExtractLinks(t *testing.T) {
	html := `<html>
<body>
  <a href="/page1">Page 1</a>
  <a href="/page2">Page 2</a>
  <a href="https://other.com/external">External</a>
  <a href="#fragment">Fragment</a>
  <a href="javascript:void(0)">JS</a>
  <a href="mailto:test@example.com">Email</a>
</body>
</html>`

	// Create a minimal crawler for link extraction
	c := &Crawler{config: &CrawlConfig{}}
	links := c.extractLinks(html, "https://example.com/docs/")

	// Should only include same-host, non-fragment, non-JS links
	expectedCount := 2 // /page1 and /page2
	if len(links) != expectedCount {
		t.Errorf("expected %d links, got %d", expectedCount, len(links))
		for _, l := range links {
			t.Logf("  - %s", l)
		}
	}

	// Check that resolved URLs are correct
	for _, link := range links {
		if link != "https://example.com/page1" && link != "https://example.com/page2" {
			t.Errorf("unexpected link: %s", link)
		}
	}
}

func TestFilterToBase(t *testing.T) {
	c := &Crawler{
		config: &CrawlConfig{
			BaseURL: "https://example.com/docs/",
		},
	}

	urls := []string{
		"https://example.com/docs/page1",
		"https://example.com/docs/guide/intro",
		"https://example.com/other/page",
		"https://other.com/docs/page",
	}

	filtered := c.filterToBase(urls)

	if len(filtered) != 2 {
		t.Errorf("expected 2 filtered URLs, got %d", len(filtered))
	}

	for _, u := range filtered {
		if u != "https://example.com/docs/page1" && u != "https://example.com/docs/guide/intro" {
			t.Errorf("unexpected URL in filtered: %s", u)
		}
	}
}

func TestNewCrawler_Defaults(t *testing.T) {
	c := NewCrawler(nil)

	if c.config == nil {
		t.Fatal("config should not be nil")
	}

	if c.config.MaxDepth != 3 {
		t.Errorf("MaxDepth = %d, want 3", c.config.MaxDepth)
	}

	if c.config.MaxPages != 100 {
		t.Errorf("MaxPages = %d, want 100", c.config.MaxPages)
	}

	if c.config.UserAgent != "mehr-library/1.0" {
		t.Errorf("UserAgent = %q, want %q", c.config.UserAgent, "mehr-library/1.0")
	}
}

func TestCrawler_Preview(t *testing.T) {
	// Create test server with sitemap
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/sitemap.xml":
			w.Header().Set("Content-Type", "application/xml")
			_, _ = w.Write([]byte(`<?xml version="1.0"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url><loc>` + "http://" + r.Host + `/docs/page1</loc></url>
  <url><loc>` + "http://" + r.Host + `/docs/page2</loc></url>
  <url><loc>` + "http://" + r.Host + `/other/page</loc></url>
</urlset>`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	c := NewCrawler(&CrawlConfig{
		BaseURL:  server.URL + "/docs/",
		MaxPages: 10,
	})

	urls, err := c.Preview(r.Context(), server.URL+"/docs/")
	if err != nil {
		t.Fatalf("Preview failed: %v", err)
	}

	// Should only include URLs within /docs/
	if len(urls) != 2 {
		t.Errorf("expected 2 URLs (filtered to base), got %d", len(urls))
		for _, u := range urls {
			t.Logf("  - %s", u)
		}
	}
}

// r is a helper to get context in tests.
var r = &http.Request{}

func init() {
	r = httptest.NewRequest(http.MethodGet, "/", nil)
}

func TestExtractRootDomain(t *testing.T) {
	tests := []struct {
		host string
		want string
	}{
		{"docs.example.com", "example.com"},
		{"api.docs.example.com", "example.com"},
		{"example.com", "example.com"},
		{"docs.example.co.uk", "example.co.uk"},
		{"api.example.co.uk", "example.co.uk"},
		{"localhost", "localhost"},
		{"127.0.0.1", "127.0.0.1"},
		{"192.168.1.1:8080", "192.168.1.1:8080"},
		{"[::1]", "[::1]"},
	}

	for _, tt := range tests {
		t.Run(tt.host, func(t *testing.T) {
			got := extractRootDomain(tt.host)
			if got != tt.want {
				t.Errorf("extractRootDomain(%q) = %q, want %q", tt.host, got, tt.want)
			}
		})
	}
}

func TestIsAllowedDomain(t *testing.T) {
	tests := []struct {
		name        string
		domainScope string
		linkHost    string
		baseHost    string
		want        bool
	}{
		// same-host (default) tests
		{"same-host: exact match", "", "docs.example.com", "docs.example.com", true},
		{"same-host: different subdomain", "", "api.example.com", "docs.example.com", false},
		{"same-host: different domain", "", "other.com", "example.com", false},

		// same-domain tests
		{"same-domain: exact match", "same-domain", "docs.example.com", "docs.example.com", true},
		{"same-domain: different subdomain", "same-domain", "api.example.com", "docs.example.com", true},
		{"same-domain: root to subdomain", "same-domain", "example.com", "docs.example.com", true},
		{"same-domain: different domain", "same-domain", "other.com", "example.com", false},
		{"same-domain: similar suffix", "same-domain", "notexample.com", "example.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Crawler{
				config: &CrawlConfig{
					DomainScope: tt.domainScope,
				},
			}

			linkURL := &url.URL{Host: tt.linkHost}
			baseURL := &url.URL{Host: tt.baseHost}

			got := c.isAllowedDomain(linkURL, baseURL)
			if got != tt.want {
				t.Errorf("isAllowedDomain(%q, %q) with scope %q = %v, want %v",
					tt.linkHost, tt.baseHost, tt.domainScope, got, tt.want)
			}
		})
	}
}

func TestContainsVersionPath(t *testing.T) {
	tests := []struct {
		path    string
		version string
		want    bool
	}{
		// Version in middle of path
		{"/docs/v24/intro", "v24", true},
		{"/api/v1.2.3/reference", "v1.2.3", true},
		{"/guide/v2/getting-started", "v2", true},

		// Version at end of path
		{"/docs/v24", "v24", true},
		{"/api/v1.0", "v1.0", true},

		// Version at start of path
		{"/v24/docs/intro", "v24", true},

		// No version
		{"/docs/latest/intro", "v24", false},
		{"/api/reference", "v1", false},
		{"/guide", "v2", false},

		// Partial matches should not match
		{"/docs/v245/intro", "v24", false},
		{"/docs/av24/intro", "v24", false},
	}

	for _, tt := range tests {
		t.Run(tt.path+"_"+tt.version, func(t *testing.T) {
			got := containsVersionPath(tt.path, tt.version)
			if got != tt.want {
				t.Errorf("containsVersionPath(%q, %q) = %v, want %v",
					tt.path, tt.version, got, tt.want)
			}
		})
	}
}

func TestDetectVersionFromPath(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"/docs/v24/intro", "v24"},
		{"/api/v1.2.3/reference", "v1.2.3"},
		{"/guide/v1.0/start", "v1.0"},
		{"/v2/docs", "v2"},

		// No version
		{"/docs/latest/intro", ""},
		{"/api/reference", ""},
		{"/guide", ""},

		// Edge cases
		{"/docs/preview/intro", ""},
		{"/api/beta/ref", ""},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := DetectVersionFromPath(tt.path)
			if got != tt.want {
				t.Errorf("DetectVersionFromPath(%q) = %q, want %q",
					tt.path, got, tt.want)
			}
		})
	}
}

func TestExtractLinks_DomainScope(t *testing.T) {
	html := `<html><body>
		<a href="/local">Local</a>
		<a href="https://docs.example.com/page1">Same Host</a>
		<a href="https://api.example.com/page2">Different Subdomain</a>
		<a href="https://other.com/page3">Different Domain</a>
	</body></html>`

	tests := []struct {
		name        string
		domainScope string
		wantCount   int
	}{
		{"same-host default", "", 2},      // /local and /page1
		{"same-domain", "same-domain", 3}, // /local, /page1, /page2
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Crawler{
				config: &CrawlConfig{
					DomainScope: tt.domainScope,
				},
			}
			links := c.extractLinks(html, "https://docs.example.com/start")

			if len(links) != tt.wantCount {
				t.Errorf("extractLinks with scope %q: got %d links, want %d",
					tt.domainScope, len(links), tt.wantCount)
				for _, l := range links {
					t.Logf("  - %s", l)
				}
			}
		})
	}
}

func TestExtractLinks_VersionFilter(t *testing.T) {
	html := `<html><body>
		<a href="/docs/v24/intro">V24 Intro</a>
		<a href="/docs/v24/guide">V24 Guide</a>
		<a href="/docs/v23/intro">V23 Intro</a>
		<a href="/docs/latest/intro">Latest Intro</a>
		<a href="/about">About</a>
	</body></html>`

	tests := []struct {
		name        string
		versionPath string
		wantCount   int
	}{
		{"no filter", "", 5},
		{"filter v24", "v24", 2},
		{"filter v23", "v23", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Crawler{
				config: &CrawlConfig{
					VersionPath: tt.versionPath,
				},
			}
			links := c.extractLinks(html, "https://example.com/docs/v24/start")

			if len(links) != tt.wantCount {
				t.Errorf("extractLinks with version %q: got %d links, want %d",
					tt.versionPath, len(links), tt.wantCount)
				for _, l := range links {
					t.Logf("  - %s", l)
				}
			}
		})
	}
}

func TestIsIPAddress(t *testing.T) {
	tests := []struct {
		host string
		want bool
	}{
		// Valid IPv4
		{"127.0.0.1", true},
		{"192.168.1.1", true},
		{"10.0.0.1", true},
		{"192.168.1.1:8080", true},
		{"0.0.0.0", true},
		{"255.255.255.255", true},

		// Valid IPv6
		{"::1", true},
		{"fe80::1", true},
		{"2001:db8::1", true},
		{"[::1]:8080", true},
		{"[2001:db8::1]:443", true},

		// Invalid IPs (should return false with net.ParseIP)
		{"999.999.999.999", false}, // Out of range - now correctly rejected
		{"256.1.1.1", false},       // Out of range octet
		{"1.2.3", false},           // Incomplete IPv4

		// Hostnames (not IPs)
		{"example.com", false},
		{"localhost", false},
		{"docs.example.com", false},
		{"my-server", false},
	}

	for _, tt := range tests {
		t.Run(tt.host, func(t *testing.T) {
			got := isIPAddress(tt.host)
			if got != tt.want {
				t.Errorf("isIPAddress(%q) = %v, want %v", tt.host, got, tt.want)
			}
		})
	}
}

func TestValidateURL(t *testing.T) {
	tests := []struct {
		name            string
		url             string
		blockPrivateIPs bool
		wantErr         bool
		errContains     string
	}{
		// Valid URLs
		{"valid http", "http://example.com/page", false, false, ""},
		{"valid https", "https://docs.example.com/v1/api", false, false, ""},
		{"valid with port", "https://example.com:8443/path", false, false, ""},

		// Invalid schemes
		{"file scheme", "file:///etc/passwd", false, true, "unsupported scheme"},
		{"ftp scheme", "ftp://example.com/file", false, true, "unsupported scheme"},
		{"javascript", "javascript:alert(1)", false, true, "unsupported scheme"},
		{"data uri", "data:text/html,<h1>test</h1>", false, true, "unsupported scheme"},

		// Private IPs (blocked)
		{"localhost blocked", "http://127.0.0.1/api", true, true, "private/internal IP"},
		{"private 10.x blocked", "http://10.0.0.1/page", true, true, "private/internal IP"},
		{"private 192.168.x blocked", "http://192.168.1.1/page", true, true, "private/internal IP"},
		{"loopback ipv6 blocked", "http://[::1]/page", true, true, "private/internal IP"},

		// Private IPs (allowed when not blocking)
		{"localhost allowed", "http://127.0.0.1/api", false, false, ""},
		{"private allowed", "http://192.168.1.1/page", false, false, ""},

		// Public IPs (always allowed)
		{"public ip", "http://8.8.8.8/dns", true, false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Crawler{
				config: &CrawlConfig{
					BlockPrivateIPs: tt.blockPrivateIPs,
				},
			}

			err := c.validateURL(tt.url)

			if tt.wantErr {
				if err == nil {
					t.Errorf("validateURL(%q) expected error containing %q, got nil", tt.url, tt.errContains)
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("validateURL(%q) error = %q, want containing %q", tt.url, err.Error(), tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("validateURL(%q) unexpected error: %v", tt.url, err)
				}
			}
		})
	}
}

func TestValidateDomainScope(t *testing.T) {
	tests := []struct {
		scope   string
		wantErr bool
	}{
		{"", false},            // Empty = default (same-host)
		{"same-host", false},   // Explicit same-host
		{"same-domain", false}, // Same domain
		{"subdomain", true},    // Invalid
		{"all", true},          // Invalid
		{"typo", true},         // Invalid
	}

	for _, tt := range tests {
		t.Run(tt.scope, func(t *testing.T) {
			err := ValidateDomainScope(tt.scope)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateDomainScope(%q) expected error, got nil", tt.scope)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateDomainScope(%q) unexpected error: %v", tt.scope, err)
				}
			}
		})
	}
}
