package localhub

import "time"

type Config struct {
	Host string `json:"host"`
	Port int    `json:"port"` // 0 means auto-pick.
}

type ReleaseAsset struct {
	Name        string    `json:"name"`
	Size        int64     `json:"size"`
	DownloadURL string    `json:"downloadUrl"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type Release struct {
	Tag         string         `json:"tag"`
	Name        string         `json:"name"`
	PublishedAt time.Time      `json:"publishedAt"`
	Assets      []ReleaseAsset `json:"assets"`
}

type InstallState struct {
	Installed   bool      `json:"installed"`
	Tag         string    `json:"tag"`
	BinaryPath  string    `json:"binaryPath"`
	InstalledAt time.Time `json:"installedAt"`
}

type RunState struct {
	Running   bool      `json:"running"`
	PID       int       `json:"pid"`
	Addr      string    `json:"addr"`
	StartedAt time.Time `json:"startedAt"`
	LogPath   string    `json:"logPath"`

	ExitedAt  time.Time `json:"exitedAt"`
	ExitError string    `json:"exitError"`
}

type DownloadState struct {
	Active         bool      `json:"active"`
	Stage          string    `json:"stage"`
	AssetName      string    `json:"assetName"`
	ExpectedSHA256 string    `json:"expectedSha256"`
	TotalBytes     int64     `json:"totalBytes"`
	DoneBytes      int64     `json:"doneBytes"`
	Message        string    `json:"message"`
	Error          string    `json:"error"`
	StartedAt      time.Time `json:"startedAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

type Snapshot struct {
	Supported bool   `json:"supported"`
	Platform  string `json:"platform"`
	Arch      string `json:"arch"`

	RootDir string `json:"rootDir"`
	BinDir  string `json:"binDir"`
	LogsDir string `json:"logsDir"`

	Config Config `json:"config"`

	LatestLoaded bool    `json:"latestLoaded"`
	LatestError  string  `json:"latestError"`
	Latest       Release `json:"latest"`

	Install  InstallState  `json:"install"`
	Run      RunState      `json:"run"`
	Download DownloadState `json:"download"`
}
