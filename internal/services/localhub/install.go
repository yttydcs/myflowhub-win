package localhub

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type downloadProgressFunc func(done int64, total int64)

func downloadFile(ctx context.Context, url string, destPath string, onProgress downloadProgressFunc) (string, int64, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if strings.TrimSpace(url) == "" {
		return "", 0, errors.New("url is required")
	}
	if strings.TrimSpace(destPath) == "" {
		return "", 0, errors.New("destPath is required")
	}

	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return "", 0, err
	}

	tmpPath := destPath + ".part"
	_ = os.Remove(tmpPath)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", 0, err
	}
	req.Header.Set("User-Agent", "myflowhub-win/localhub")

	client := &http.Client{
		Timeout: 0,
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", 0, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", 0, fmt.Errorf("download failed: status %d", resp.StatusCode)
	}

	f, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return "", 0, err
	}
	defer func() { _ = f.Close() }()

	total := resp.ContentLength
	hasher := sha256.New()
	var done int64
	lastReport := time.Time{}

	buf := make([]byte, 64<<10)
	for {
		if err := ctx.Err(); err != nil {
			return "", done, err
		}
		n, rerr := resp.Body.Read(buf)
		if n > 0 {
			chunk := buf[:n]
			if _, err := f.Write(chunk); err != nil {
				return "", done, err
			}
			if _, err := hasher.Write(chunk); err != nil {
				return "", done, err
			}
			done += int64(n)

			now := time.Now()
			if onProgress != nil && (lastReport.IsZero() || now.Sub(lastReport) >= 200*time.Millisecond) {
				lastReport = now
				onProgress(done, total)
			}
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			return "", done, rerr
		}
	}
	if onProgress != nil {
		onProgress(done, total)
	}

	sum := hex.EncodeToString(hasher.Sum(nil))
	if err := f.Close(); err != nil {
		return "", done, err
	}
	if err := os.Remove(destPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", done, err
	}
	if err := os.Rename(tmpPath, destPath); err != nil {
		return "", done, err
	}
	return sum, done, nil
}

func readRemoteText(ctx context.Context, url string) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "myflowhub-win/localhub")
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("download failed: status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func parseChecksums(text string) map[string]string {
	out := map[string]string{}
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		sum := strings.ToLower(strings.TrimSpace(fields[0]))
		name := strings.TrimSpace(fields[len(fields)-1])
		if sum == "" || name == "" {
			continue
		}
		out[name] = sum
	}
	return out
}

func extractBinaryFromZip(zipPath string, wantFileName string, destPath string) error {
	if strings.TrimSpace(zipPath) == "" {
		return errors.New("zipPath is required")
	}
	if strings.TrimSpace(wantFileName) == "" {
		return errors.New("wantFileName is required")
	}
	if strings.TrimSpace(destPath) == "" {
		return errors.New("destPath is required")
	}

	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer func() { _ = zr.Close() }()

	var target *zip.File
	for _, f := range zr.File {
		if f == nil {
			continue
		}
		name := filepath.Base(filepath.Clean(f.Name))
		if name == wantFileName {
			target = f
			break
		}
	}
	if target == nil {
		return fmt.Errorf("binary %s not found in zip", wantFileName)
	}

	r, err := target.Open()
	if err != nil {
		return err
	}
	defer func() { _ = r.Close() }()

	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return err
	}

	tmpPath := destPath + ".tmp"
	_ = os.Remove(tmpPath)
	f, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, r); err != nil {
		_ = f.Close()
		_ = os.Remove(tmpPath)
		return err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}

	if runtime.GOOS != "windows" {
		_ = os.Chmod(tmpPath, 0o755)
	}
	if err := os.Remove(destPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		_ = os.Remove(tmpPath)
		return err
	}
	return os.Rename(tmpPath, destPath)
}
