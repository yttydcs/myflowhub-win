package localhub

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/yttydcs/myflowhub-win/internal/services/logs"
	"github.com/yttydcs/myflowhub-win/internal/storage"
)

const (
	defaultHost = "127.0.0.1"
	defaultPort = 9000

	keyHost         = "localhub.host"
	keyPort         = "localhub.port"
	keyInstalledTag = "localhub.installed.tag"
	keyInstalledAt  = "localhub.installed.at"
)

type LocalHubService struct {
	mu    sync.Mutex
	store *storage.Store
	logs  *logs.LogService

	cfg Config

	latestLoaded bool
	latestErr    string
	latest       Release

	install InstallState
	run     RunState
	dl      DownloadState

	cmd     *exec.Cmd
	waitCh  chan struct{}
	waitErr error
}

func New(store *storage.Store, logsSvc *logs.LogService) *LocalHubService {
	s := &LocalHubService{store: store, logs: logsSvc}
	s.mu.Lock()
	s.loadConfigLocked()
	s.refreshInstallLocked()
	s.mu.Unlock()
	return s
}

func (s *LocalHubService) Snapshot() Snapshot {
	s.mu.Lock()
	s.refreshInstallLocked()
	snap := Snapshot{
		Supported:    isSupported(),
		Platform:     runtime.GOOS,
		Arch:         runtime.GOARCH,
		RootDir:      s.rootDirLocked(),
		BinDir:       s.binDirLocked(),
		LogsDir:      s.logsDirLocked(),
		Config:       s.cfg,
		LatestLoaded: s.latestLoaded,
		LatestError:  s.latestErr,
		Latest:       s.latest,
		Install:      s.install,
		Run:          s.run,
		Download:     s.dl,
	}
	s.mu.Unlock()
	return snap
}

func (s *LocalHubService) SaveConfig(cfg Config) (Config, error) {
	cfg.Host = strings.TrimSpace(cfg.Host)
	if cfg.Host == "" {
		cfg.Host = defaultHost
	}
	if cfg.Port < 0 || cfg.Port > 65535 {
		return Config{}, errors.New("port must be 0..65535")
	}

	s.mu.Lock()
	s.cfg = cfg
	if err := s.saveConfigLocked(); err != nil {
		s.mu.Unlock()
		return Config{}, err
	}
	out := s.cfg
	s.mu.Unlock()
	return out, nil
}

func (s *LocalHubService) RefreshLatest() (Release, error) {
	if !isSupported() {
		return Release{}, fmt.Errorf("unsupported platform: %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	rel, err := fetchLatestRelease(ctx)

	s.mu.Lock()
	if err != nil {
		s.latestLoaded = false
		s.latestErr = err.Error()
		s.latest = Release{}
	} else {
		s.latestLoaded = true
		s.latestErr = ""
		s.latest = rel
	}
	s.mu.Unlock()
	return rel, err
}

func (s *LocalHubService) InstallLatest() (InstallState, error) {
	if !isSupported() {
		return InstallState{}, fmt.Errorf("unsupported platform: %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	s.mu.Lock()
	if s.cmd != nil || s.run.Running {
		s.mu.Unlock()
		return InstallState{}, errors.New("hub_server is running (stop it before installing)")
	}
	if s.dl.Active {
		s.mu.Unlock()
		return InstallState{}, errors.New("download already in progress")
	}
	s.dl = DownloadState{
		Active:    true,
		Stage:     "fetch_latest",
		StartedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.dl.Active = false
		s.dl.UpdatedAt = time.Now()
		s.mu.Unlock()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	logf := func(level string, format string, args ...any) {
		if s.logs != nil {
			s.logs.Appendf(level, format, args...)
		}
	}

	rel, err := fetchLatestRelease(ctx)
	if err != nil {
		s.setDownloadError("fetch_latest", err)
		return InstallState{}, err
	}
	s.mu.Lock()
	s.latestLoaded = true
	s.latestErr = ""
	s.latest = rel
	s.mu.Unlock()

	zipName, err := platformZipAsset()
	if err != nil {
		s.setDownloadError("platform", err)
		return InstallState{}, err
	}

	zipURL := ""
	checksumURL := ""
	for _, asset := range rel.Assets {
		switch asset.Name {
		case zipName:
			zipURL = asset.DownloadURL
		case "checksums.txt":
			checksumURL = asset.DownloadURL
		}
	}
	if zipURL == "" {
		err := fmt.Errorf("missing release asset: %s", zipName)
		s.setDownloadError("resolve_assets", err)
		return InstallState{}, err
	}
	if checksumURL == "" {
		err := errors.New("missing release asset: checksums.txt")
		s.setDownloadError("resolve_assets", err)
		return InstallState{}, err
	}

	s.mu.Lock()
	s.dl.Stage = "download_checksums"
	s.dl.AssetName = "checksums.txt"
	s.dl.UpdatedAt = time.Now()
	s.mu.Unlock()

	checksumsText, err := readRemoteText(ctx, checksumURL)
	if err != nil {
		s.setDownloadError("download_checksums", err)
		return InstallState{}, err
	}
	checksums := parseChecksums(checksumsText)
	expectSum := strings.ToLower(strings.TrimSpace(checksums[zipName]))
	if expectSum == "" {
		err := fmt.Errorf("checksums.txt missing entry for %s", zipName)
		s.setDownloadError("parse_checksums", err)
		return InstallState{}, err
	}

	s.mu.Lock()
	s.dl.Stage = "download_zip"
	s.dl.AssetName = zipName
	s.dl.ExpectedSHA256 = expectSum
	s.dl.TotalBytes = 0
	s.dl.DoneBytes = 0
	s.dl.UpdatedAt = time.Now()
	s.mu.Unlock()

	downloadsDir := filepath.Join(s.rootDir(), "downloads")
	destZip := filepath.Join(downloadsDir, zipName)

	logf("info", "localhub download: %s (%s)", zipName, rel.Tag)
	gotSum, _, err := downloadFile(ctx, zipURL, destZip, func(done int64, total int64) {
		s.mu.Lock()
		s.dl.DoneBytes = done
		s.dl.TotalBytes = total
		s.dl.UpdatedAt = time.Now()
		s.mu.Unlock()
	})
	if err != nil {
		s.setDownloadError("download_zip", err)
		return InstallState{}, err
	}
	if !strings.EqualFold(gotSum, expectSum) {
		err := fmt.Errorf("checksum mismatch for %s: expected %s got %s", zipName, expectSum, gotSum)
		s.setDownloadError("verify_checksum", err)
		return InstallState{}, err
	}

	s.mu.Lock()
	s.dl.Stage = "extract"
	s.dl.Message = "Extracting binary..."
	s.dl.UpdatedAt = time.Now()
	s.mu.Unlock()

	binName := platformBinaryName()
	destBin := filepath.Join(s.binDir(), binName)
	if err := extractBinaryFromZip(destZip, binName, destBin); err != nil {
		s.setDownloadError("extract", err)
		return InstallState{}, err
	}

	now := time.Now()
	s.mu.Lock()
	s.install = InstallState{
		Installed:   true,
		Tag:         rel.Tag,
		BinaryPath:  destBin,
		InstalledAt: now,
	}
	if err := s.saveInstalledLocked(now, rel.Tag); err != nil && s.logs != nil {
		s.logs.Appendf("warn", "localhub save installed state warning: %v", err)
	}
	s.dl.Stage = "done"
	s.dl.Message = "Installed."
	s.dl.UpdatedAt = time.Now()
	out := s.install
	s.mu.Unlock()

	logf("info", "localhub installed: %s (%s)", destBin, rel.Tag)
	return out, nil
}

func (s *LocalHubService) Start() (RunState, error) {
	if !isSupported() {
		return RunState{}, fmt.Errorf("unsupported platform: %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	s.mu.Lock()
	if s.run.Running && s.cmd != nil {
		out := s.run
		s.mu.Unlock()
		return out, nil
	}
	s.refreshInstallLocked()
	if !s.install.Installed || strings.TrimSpace(s.install.BinaryPath) == "" {
		s.mu.Unlock()
		return RunState{}, errors.New("hub_server not installed (install latest first)")
	}
	bin := s.install.BinaryPath
	cfg := s.cfg
	s.mu.Unlock()

	port, changed, err := pickPort(cfg.Host, cfg.Port)
	if err != nil {
		return RunState{}, err
	}
	addr := netJoin(cfg.Host, port)
	if changed && s.logs != nil {
		s.logs.Appendf("warn", "localhub port adjusted: requested=%d actual=%d", cfg.Port, port)
	}

	logPath := filepath.Join(s.logsDir(), fmt.Sprintf("hub_server_%s.log", time.Now().Format("20060102_150405")))
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		return RunState{}, err
	}
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return RunState{}, err
	}

	cmd := exec.Command(bin, "-addr", addr)
	cmd.Dir = filepath.Dir(bin)
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	configureDetached(cmd)
	if err := cmd.Start(); err != nil {
		_ = logFile.Close()
		return RunState{}, err
	}

	waitCh := make(chan struct{})

	s.mu.Lock()
	s.cmd = cmd
	s.waitCh = waitCh
	s.run = RunState{
		Running:   true,
		PID:       cmd.Process.Pid,
		Addr:      addr,
		StartedAt: time.Now(),
		LogPath:   logPath,
	}
	out := s.run
	s.mu.Unlock()

	if s.logs != nil {
		s.logs.Appendf("info", "localhub started pid=%d addr=%s", cmd.Process.Pid, addr)
	}

	go func() {
		err := cmd.Wait()
		_ = logFile.Close()
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.cmd == cmd {
			s.cmd = nil
		}
		s.waitErr = err
		s.run.Running = false
		s.run.ExitedAt = time.Now()
		if err != nil {
			s.run.ExitError = err.Error()
		}
		close(waitCh)
	}()

	return out, nil
}

func (s *LocalHubService) Stop() error {
	s.mu.Lock()
	cmd := s.cmd
	waitCh := s.waitCh
	s.mu.Unlock()

	if cmd == nil || cmd.Process == nil {
		return errors.New("hub_server not running")
	}

	if runtime.GOOS == "windows" {
		_ = cmd.Process.Kill()
	} else {
		_ = cmd.Process.Signal(syscall.SIGTERM)
		if waitCh != nil {
			select {
			case <-waitCh:
			case <-time.After(3 * time.Second):
				_ = cmd.Process.Kill()
				select {
				case <-waitCh:
				case <-time.After(2 * time.Second):
				}
			}
		} else {
			_ = cmd.Process.Kill()
		}
	}

	if s.logs != nil {
		s.logs.Append("info", "localhub stopped")
	}
	return nil
}

func (s *LocalHubService) Restart() (RunState, error) {
	_ = s.Stop()
	return s.Start()
}

func (s *LocalHubService) rootDir() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.rootDirLocked()
}

func (s *LocalHubService) binDir() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.binDirLocked()
}

func (s *LocalHubService) logsDir() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.logsDirLocked()
}

func (s *LocalHubService) rootDirLocked() string {
	base := ""
	if s.store != nil {
		base = s.store.BaseDir()
	}
	if strings.TrimSpace(base) == "" {
		if dir, err := os.UserConfigDir(); err == nil && strings.TrimSpace(dir) != "" {
			base = filepath.Join(dir, "myflowhub-win")
		}
	}
	if strings.TrimSpace(base) == "" {
		base = "."
	}
	return filepath.Join(base, "localhub")
}

func (s *LocalHubService) binDirLocked() string {
	return filepath.Join(s.rootDirLocked(), "bin")
}

func (s *LocalHubService) logsDirLocked() string {
	return filepath.Join(s.rootDirLocked(), "logs")
}

func (s *LocalHubService) loadConfigLocked() {
	cfg := Config{Host: defaultHost, Port: defaultPort}
	if s.store != nil {
		host := s.store.GetString("", keyHost, cfg.Host)
		port := s.store.GetInt("", keyPort, cfg.Port)
		cfg.Host = strings.TrimSpace(host)
		cfg.Port = port
	}
	cfg.Host = strings.TrimSpace(cfg.Host)
	if cfg.Host == "" {
		cfg.Host = defaultHost
	}
	if cfg.Port < 0 || cfg.Port > 65535 {
		cfg.Port = defaultPort
	}
	s.cfg = cfg
}

func (s *LocalHubService) saveConfigLocked() error {
	if s.store == nil {
		return nil
	}
	if err := s.store.SetString("", keyHost, s.cfg.Host); err != nil {
		return err
	}
	if err := s.store.SetInt("", keyPort, s.cfg.Port); err != nil {
		return err
	}
	return nil
}

func (s *LocalHubService) refreshInstallLocked() {
	bin := filepath.Join(s.binDirLocked(), platformBinaryName())
	info, err := os.Stat(bin)
	installed := err == nil && info != nil && !info.IsDir()

	tag := ""
	var installedAt time.Time
	if s.store != nil {
		tag = s.store.GetString("", keyInstalledTag, "")
		if raw := strings.TrimSpace(s.store.GetString("", keyInstalledAt, "")); raw != "" {
			if t, err := time.Parse(time.RFC3339, raw); err == nil {
				installedAt = t
			}
		}
	}
	s.install = InstallState{
		Installed:   installed,
		Tag:         strings.TrimSpace(tag),
		BinaryPath:  bin,
		InstalledAt: installedAt,
	}
}

func (s *LocalHubService) saveInstalledLocked(at time.Time, tag string) error {
	if s.store == nil {
		return nil
	}
	if err := s.store.SetString("", keyInstalledTag, strings.TrimSpace(tag)); err != nil {
		return err
	}
	if err := s.store.SetString("", keyInstalledAt, at.UTC().Format(time.RFC3339)); err != nil {
		return err
	}
	return nil
}

func (s *LocalHubService) setDownloadError(stage string, err error) {
	s.mu.Lock()
	s.dl.Stage = stage
	s.dl.Error = err.Error()
	s.dl.UpdatedAt = time.Now()
	s.mu.Unlock()
	if s.logs != nil {
		s.logs.Appendf("error", "localhub %s: %v", stage, err)
	}
}

func platformBinaryName() string {
	if runtime.GOOS == "windows" {
		return "hub_server.exe"
	}
	return "hub_server"
}

func platformZipAsset() (string, error) {
	switch runtime.GOOS {
	case "windows":
		if runtime.GOARCH != "amd64" {
			return "", fmt.Errorf("unsupported arch: %s", runtime.GOARCH)
		}
		return "hub_server_windows_amd64.zip", nil
	case "linux":
		if runtime.GOARCH != "amd64" {
			return "", fmt.Errorf("unsupported arch: %s", runtime.GOARCH)
		}
		return "hub_server_linux_amd64.zip", nil
	default:
		return "", fmt.Errorf("unsupported platform: %s/%s", runtime.GOOS, runtime.GOARCH)
	}
}

func isSupported() bool {
	if runtime.GOARCH != "amd64" {
		return false
	}
	return runtime.GOOS == "windows" || runtime.GOOS == "linux"
}

func netJoin(host string, port int) string {
	host = strings.TrimSpace(host)
	if host == "" {
		return ":" + strconv.Itoa(port)
	}
	return net.JoinHostPort(host, strconv.Itoa(port))
}
