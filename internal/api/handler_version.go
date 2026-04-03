package api

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

type versionResponse struct {
	Version string         `json:"version"`
	Mode    string         `json:"mode"`
	Latest  *latestRelease `json:"latest"`
}

type latestRelease struct {
	Version     string `json:"version"`
	URL         string `json:"url"`
	PublishedAt string `json:"publishedAt"`
}

var (
	releaseCacheMu   sync.Mutex
	releaseCacheTime time.Time
	releaseCacheData *latestRelease
)

const (
	releaseCacheTTL = 1 * time.Hour
	releaseRepo     = "maxzhang666/ops-tunnel"
)

func (s *Server) getVersion(w http.ResponseWriter, r *http.Request) {
	latest := fetchLatestRelease()
	writeJSON(w, http.StatusOK, versionResponse{
		Version: s.cfg.Version,
		Mode:    s.cfg.Mode,
		Latest:  latest,
	})
}

func fetchLatestRelease() *latestRelease {
	releaseCacheMu.Lock()
	defer releaseCacheMu.Unlock()

	if time.Since(releaseCacheTime) < releaseCacheTTL && releaseCacheData != nil {
		return releaseCacheData
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get("https://api.github.com/repos/" + releaseRepo + "/releases/latest")
	if err != nil {
		return releaseCacheData
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return releaseCacheData
	}

	var ghRelease struct {
		TagName     string `json:"tag_name"`
		HTMLURL     string `json:"html_url"`
		PublishedAt string `json:"published_at"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&ghRelease); err != nil {
		return releaseCacheData
	}

	ver := ghRelease.TagName
	if len(ver) > 0 && ver[0] == 'v' {
		ver = ver[1:]
	}

	releaseCacheData = &latestRelease{
		Version:     ver,
		URL:         ghRelease.HTMLURL,
		PublishedAt: ghRelease.PublishedAt,
	}
	releaseCacheTime = time.Now()
	return releaseCacheData
}
