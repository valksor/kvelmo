package update

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/google/go-github/v67/github"
	"golang.org/x/mod/semver"
	"golang.org/x/oauth2"
)

// Checker checks for available updates from GitHub releases.
type Checker struct {
	ghClient *github.Client
	owner    string
	repo     string
}

// NewChecker creates a new update checker.
// If token is empty, the client will make unauthenticated requests (subject to rate limits).
func NewChecker(ctx context.Context, token, owner, repo string) *Checker {
	var tc *github.Client
	if token != "" {
		ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
		httpClient := oauth2.NewClient(ctx, ts)
		tc = github.NewClient(httpClient)
	} else {
		tc = github.NewClient(nil)
	}

	// Set default owner/repo if not provided
	owner = cmp.Or(owner, "valksor")
	repo = cmp.Or(repo, "go-mehrhof")

	return &Checker{
		ghClient: tc,
		owner:    owner,
		repo:     repo,
	}
}

// Check looks for available updates and returns the status.
// It returns ErrNoUpdateAvailable if the current version is up to date.
func (c *Checker) Check(ctx context.Context, opts CheckOptions) (*UpdateStatus, error) {
	// Set owner/repo from options if provided
	if opts.Owner != "" {
		c.owner = opts.Owner
	}
	if opts.Repo != "" {
		c.repo = opts.Repo
	}

	var latestRelease *github.RepositoryRelease
	var err error

	if opts.TargetTag != "" {
		latestRelease, err = c.findReleaseByTag(ctx, opts.TargetTag)
		if err != nil {
			return nil, err
		}
	} else {
		latestRelease, err = c.findLatestRelease(ctx, opts.IncludeNightly)
		if err != nil {
			return nil, err
		}
	}

	if latestRelease == nil {
		return nil, errors.New("no suitable release found")
	}

	// Normalize version strings for comparison
	latestVersion := strings.TrimPrefix(latestRelease.GetTagName(), "v")
	currentVersion := strings.TrimPrefix(opts.CurrentVersion, "v")

	// For explicit targets, install the requested version even if already installed.
	if opts.TargetTag == "" {
		// For dev/none builds, always offer installing the latest release.
		if opts.CurrentVersion != "dev" && opts.CurrentVersion != "none" {
			// If current is same or newer than latest, no update needed.
			if currentVersion == latestVersion || versionNewer(currentVersion, latestVersion) {
				return &UpdateStatus{
					CurrentVersion: opts.CurrentVersion,
					LatestVersion:  latestRelease.GetTagName(),
					IsNewer:        false,
					IsPreRelease:   latestRelease.GetPrerelease(),
				}, ErrNoUpdateAvailable
			}
		}
	}

	// Find the matching asset for this platform
	expectedAsset := fmt.Sprintf("mehr-%s-%s", runtime.GOOS, runtime.GOARCH)
	var assetURL string
	var assetSize int64
	var checksumsURL string

	for _, asset := range latestRelease.Assets {
		name := asset.GetName()
		switch name {
		case expectedAsset:
			assetURL = asset.GetBrowserDownloadURL()
			assetSize = int64(asset.GetSize())
		case "checksums.txt":
			checksumsURL = asset.GetBrowserDownloadURL()
		}
	}

	if assetURL == "" {
		return nil, fmt.Errorf("%w: %s for %s/%s", ErrAssetNotFound, expectedAsset, runtime.GOOS, runtime.GOARCH)
	}

	status := &UpdateStatus{
		CurrentVersion: opts.CurrentVersion,
		LatestVersion:  latestRelease.GetTagName(),
		AssetName:      expectedAsset,
		AssetURL:       assetURL,
		AssetSize:      assetSize,
		IsNewer:        true,
		IsPreRelease:   latestRelease.GetPrerelease(),
		ReleaseURL:     latestRelease.GetHTMLURL(),
		ReleaseNotes:   latestRelease.GetBody(),
		Checksum:       "", // Will be populated by fetching checksums
	}

	// Fetch checksums if available
	if checksumsURL != "" {
		status.Checksum, _ = c.fetchChecksum(ctx, checksumsURL, expectedAsset)
	}

	return status, nil
}

func (c *Checker) findLatestRelease(ctx context.Context, includeNightly bool) (*github.RepositoryRelease, error) {
	releases, _, err := c.ghClient.Repositories.ListReleases(ctx, c.owner, c.repo, &github.ListOptions{
		PerPage: 30,
	})
	if err != nil {
		return nil, fmt.Errorf("fetch releases: %w", err)
	}

	if len(releases) == 0 {
		return nil, errors.New("no releases found")
	}

	for _, r := range releases {
		if r.GetDraft() {
			continue
		}
		if !includeNightly && r.GetPrerelease() {
			continue
		}

		return r, nil
	}

	return nil, errors.New("no suitable release found")
}

func (c *Checker) findReleaseByTag(ctx context.Context, tag string) (*github.RepositoryRelease, error) {
	tags := []string{tag}
	// Accept either "vX.Y.Z" or "X.Y.Z" for convenience.
	if strings.HasPrefix(tag, "v") {
		tags = append(tags, strings.TrimPrefix(tag, "v"))
	} else if tag != "nightly" {
		tags = append(tags, "v"+tag)
	}

	for _, candidate := range tags {
		release, resp, err := c.ghClient.Repositories.GetReleaseByTag(ctx, c.owner, c.repo, candidate)
		if err == nil {
			return release, nil
		}
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			continue
		}

		return nil, fmt.Errorf("get release by tag %q: %w", candidate, err)
	}

	return nil, fmt.Errorf("%w: %q", ErrReleaseNotFound, tag)
}

// fetchChecksum downloads and parses the checksums file to find the checksum for the given asset.
func (c *Checker) fetchChecksum(ctx context.Context, url, assetName string) (string, error) {
	// We can't use ghClient directly for this, we need to make a raw HTTP request
	// But we can let the downloader handle this - just return empty for now
	// The downloader will fetch and parse the checksums file
	return "", nil
}

// versionNewer compares two version strings and returns true if a is newer than b.
// Uses golang.org/x/mod/semver for proper semantic version comparison.
func versionNewer(a, b string) bool {
	// semver.Compare requires versions to start with "v"
	if !strings.HasPrefix(a, "v") {
		a = "v" + a
	}
	if !strings.HasPrefix(b, "v") {
		b = "v" + b
	}

	return semver.Compare(a, b) > 0
}

// ReleaseInfoFromGitHub converts a GitHub release to our ReleaseInfo type.
func ReleaseInfoFromGitHub(gh *github.RepositoryRelease) *ReleaseInfo {
	assets := make([]Asset, 0, len(gh.Assets))
	for _, a := range gh.Assets {
		assets = append(assets, Asset{
			Name: a.GetName(),
			URL:  a.GetBrowserDownloadURL(),
			Size: int64(a.GetSize()),
		})
	}

	var publishedAt time.Time
	if gh.PublishedAt != nil {
		publishedAt = gh.PublishedAt.Time
	}

	return &ReleaseInfo{
		TagName:     gh.GetTagName(),
		Name:        gh.GetName(),
		PreRelease:  gh.GetPrerelease(),
		PublishedAt: publishedAt,
		HTMLURL:     gh.GetHTMLURL(),
		Body:        gh.GetBody(),
		Assets:      assets,
	}
}
