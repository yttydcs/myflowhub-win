package localhub

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	githubOwner = "yttydcs"
	githubRepo  = "myflowhub-server"
)

type githubRelease struct {
	TagName     string        `json:"tag_name"`
	Name        string        `json:"name"`
	PublishedAt time.Time     `json:"published_at"`
	Assets      []githubAsset `json:"assets"`
	Message     string        `json:"message"`
}

type githubAsset struct {
	Name               string    `json:"name"`
	BrowserDownloadURL string    `json:"browser_download_url"`
	Size               int64     `json:"size"`
	UpdatedAt          time.Time `json:"updated_at"`
}

func fetchLatestRelease(ctx context.Context) (Release, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", githubOwner, githubRepo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return Release{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "myflowhub-win/localhub")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return Release{}, err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return Release{}, err
	}

	if resp.StatusCode == http.StatusNotFound {
		return Release{}, errors.New("no GitHub Releases found for hub_server (push a v* tag in myflowhub-server to create one)")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var msg githubRelease
		_ = json.Unmarshal(body, &msg)
		if strings.TrimSpace(msg.Message) != "" {
			return Release{}, fmt.Errorf("GitHub API error (%d): %s", resp.StatusCode, msg.Message)
		}
		return Release{}, fmt.Errorf("GitHub API error (%d)", resp.StatusCode)
	}

	var gr githubRelease
	if err := json.Unmarshal(body, &gr); err != nil {
		return Release{}, err
	}
	if strings.TrimSpace(gr.TagName) == "" {
		return Release{}, errors.New("GitHub latest release missing tag_name")
	}

	out := Release{
		Tag:         strings.TrimSpace(gr.TagName),
		Name:        strings.TrimSpace(gr.Name),
		PublishedAt: gr.PublishedAt,
		Assets:      make([]ReleaseAsset, 0, len(gr.Assets)),
	}
	for _, asset := range gr.Assets {
		name := strings.TrimSpace(asset.Name)
		if name == "" || strings.TrimSpace(asset.BrowserDownloadURL) == "" {
			continue
		}
		out.Assets = append(out.Assets, ReleaseAsset{
			Name:        name,
			Size:        asset.Size,
			DownloadURL: asset.BrowserDownloadURL,
			UpdatedAt:   asset.UpdatedAt,
		})
	}
	return out, nil
}
