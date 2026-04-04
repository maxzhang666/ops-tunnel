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
	WsPort  int            `json:"wsPort,omitempty"`
	Latest  *latestRelease `json:"latest"`
}

type releaseAsset struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type latestRelease struct {
	Version     string         `json:"version"`
	URL         string         `json:"url"`
	PublishedAt string         `json:"publishedAt"`
	Assets      []releaseAsset `json:"assets,omitempty"`
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
	resp := versionResponse{
		Version: s.cfg.Version,
		Mode:    s.cfg.Mode,
		WsPort:  s.cfg.WsPort,
		Latest:  latest,
	}
	writeJSON(w, http.StatusOK, resp)
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
		Assets      []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&ghRelease); err != nil {
		return releaseCacheData
	}

	ver := ghRelease.TagName
	if len(ver) > 0 && ver[0] == 'v' {
		ver = ver[1:]
	}

	assets := make([]releaseAsset, 0, len(ghRelease.Assets))
	for _, a := range ghRelease.Assets {
		assets = append(assets, releaseAsset{Name: a.Name, URL: a.BrowserDownloadURL})
	}

	releaseCacheData = &latestRelease{
		Version:     ver,
		URL:         ghRelease.HTMLURL,
		PublishedAt: ghRelease.PublishedAt,
		Assets:      assets,
	}
	releaseCacheTime = time.Now()
	return releaseCacheData
}
