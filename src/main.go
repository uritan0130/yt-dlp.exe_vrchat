package main

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	ytDLPURL       = "https://github.com/yt-dlp/yt-dlp/releases/download/2026.08.19/yt-dlp.exe"
	ytDLPSHA256    = "66674953fe251b89f4d08c5f0e35e0728679bd67ab3d7d05c0562af101dd3e7a"
	denoURL        = "https://github.com/denoland/deno/releases/download/v2.9.5/deno-x86_64-pc-windows-msvc.zip"
	denoSHA512     = "7c4b0701e85f105b4ad000a8cab575203c5fa6e95adc47d3f14df87b8b11f90b8d2704de824d61368b4571a03ac7ef83d49dd176fee8713bfc8c9270c4a35b92"
	formatSelector = "best[protocol^=m3u8][vcodec^=avc][acodec^=mp4a][height<=?4096]/best[protocol^=m3u8][vcodec!=none][acodec!=none][height<=?4096]/best[ext=mp4][vcodec!=none][acodec!=none][height<=?4096]/best[vcodec!=none][acodec!=none][height<=?4096]"
)

type attempt struct {
	name          string
	extractorArgs string
}

var attempts = []attempt{
	{name: "web_embedded_1", extractorArgs: "youtube:player_client=web_embedded"},
	{name: "web_embedded_2", extractorArgs: "youtube:player_client=web_embedded"},
	{name: "web_embedded_3", extractorArgs: "youtube:player_client=web_embedded"},
	{name: "android", extractorArgs: "youtube:player_client=android"},
	{name: "default"},
}

func quoteArgs(args []string) string {
	q := make([]string, 0, len(args))
	for _, a := range args {
		if strings.ContainsAny(a, " \t\"") {
			q = append(q, fmt.Sprintf("%q", a))
		} else {
			q = append(q, a)
		}
	}
	return strings.Join(q, " ")
}

func logLine(dir, msg string) {
	p := filepath.Join(dir, "yt-dlp-onefile.log")
	f, err := os.OpenFile(p, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "%s %s\n", time.Now().Format("2006-01-02 15:04:05"), msg)
}

func cacheDir() string {
	if p := os.Getenv("LOCALAPPDATA"); p != "" {
		return filepath.Join(p, "VRChatYTDLP-OneFile", "bin")
	}
	if p, err := os.UserCacheDir(); err == nil && p != "" {
		return filepath.Join(p, "VRChatYTDLP-OneFile", "bin")
	}
	return filepath.Join(os.TempDir(), "VRChatYTDLP-OneFile", "bin")
}

func hashSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func hashSHA512(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha512.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	tmp := dst + ".tmp"
	out, err := os.Create(tmp)
	if err != nil {
		return err
	}
	_, cpErr := io.Copy(out, in)
	closeErr := out.Close()
	if cpErr != nil {
		os.Remove(tmp)
		return cpErr
	}
	if closeErr != nil {
		os.Remove(tmp)
		return closeErr
	}
	os.Remove(dst)
	return os.Rename(tmp, dst)
}

func downloadFile(rawURL, dst string) error {
	client := &http.Client{Timeout: 90 * time.Second}
	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "VRChatYTDLP-OneFile/1.0")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	tmp := dst + ".download"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	_, cpErr := io.Copy(f, resp.Body)
	closeErr := f.Close()
	if cpErr != nil {
		os.Remove(tmp)
		return cpErr
	}
	if closeErr != nil {
		os.Remove(tmp)
		return closeErr
	}
	os.Remove(dst)
	return os.Rename(tmp, dst)
}

func extractDeno(zipPath, dst string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()
	for _, f := range r.File {
		if strings.EqualFold(filepath.Base(f.Name), "deno.exe") {
			rc, err := f.Open()
			if err != nil {
				return err
			}
			defer rc.Close()
			if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
				return err
			}
			tmp := dst + ".tmp"
			out, err := os.Create(tmp)
			if err != nil {
				return err
			}
			_, cpErr := io.Copy(out, rc)
			closeErr := out.Close()
			if cpErr != nil {
				os.Remove(tmp)
				return cpErr
			}
			if closeErr != nil {
				os.Remove(tmp)
				return closeErr
			}
			os.Remove(dst)
			return os.Rename(tmp, dst)
		}
	}
	return fmt.Errorf("deno.exe not found in archive")
}

func ensureDeps(exeDir, logDir string) (string, string, error) {
	cdir := cacheDir()
	if err := os.MkdirAll(cdir, 0755); err != nil {
		return "", "", err
	}
	real := filepath.Join(cdir, "real-yt-dlp-2026.08.19.exe")
	deno := filepath.Join(cdir, "deno-2.9.5.exe")

	// yt-dlp: accept cached file only if it matches the official release hash.
	ok := false
	if h, err := hashSHA256(real); err == nil && strings.EqualFold(h, ytDLPSHA256) {
		ok = true
	}
	if !ok {
		beside := filepath.Join(exeDir, "real-yt-dlp.exe")
		if h, err := hashSHA256(beside); err == nil && strings.EqualFold(h, ytDLPSHA256) {
			logLine(logDir, "BOOTSTRAP copying verified real-yt-dlp.exe to cache")
			if err := copyFile(beside, real); err != nil {
				return "", "", err
			}
		} else {
			logLine(logDir, "BOOTSTRAP downloading official yt-dlp 2026.08.19")
			if err := downloadFile(ytDLPURL, real); err != nil {
				return "", "", fmt.Errorf("yt-dlp download failed: %w", err)
			}
			h, err := hashSHA256(real)
			if err != nil || !strings.EqualFold(h, ytDLPSHA256) {
				os.Remove(real)
				return "", "", fmt.Errorf("yt-dlp hash verification failed")
			}
		}
	}

	// Deno: prefer the already-working local deno.exe if present. Otherwise use official v2.9.5.
	if st, err := os.Stat(deno); err != nil || st.Size() < 1024*1024 {
		beside := filepath.Join(exeDir, "deno.exe")
		if st2, err2 := os.Stat(beside); err2 == nil && st2.Size() > 1024*1024 {
			logLine(logDir, "BOOTSTRAP copying existing deno.exe to cache")
			if err := copyFile(beside, deno); err != nil {
				return "", "", err
			}
		} else {
			zipPath := filepath.Join(cdir, "deno-2.9.5.zip")
			logLine(logDir, "BOOTSTRAP downloading official Deno 2.9.5")
			if err := downloadFile(denoURL, zipPath); err != nil {
				return "", "", fmt.Errorf("Deno download failed: %w", err)
			}
			h, err := hashSHA512(zipPath)
			if err != nil || !strings.EqualFold(h, denoSHA512) {
				os.Remove(zipPath)
				return "", "", fmt.Errorf("Deno hash verification failed")
			}
			if err := extractDeno(zipPath, deno); err != nil {
				return "", "", err
			}
			os.Remove(zipPath)
		}
	}
	return real, deno, nil
}

func childEnv(denoPath string) []string {
	d := filepath.Dir(denoPath)
	path := os.Getenv("PATH")
	return append(os.Environ(), "PATH="+d+";"+path)
}

func hasHTTPURL(args []string) bool {
	for _, a := range args {
		l := strings.ToLower(a)
		if strings.HasPrefix(l, "http://") || strings.HasPrefix(l, "https://") {
			return true
		}
	}
	return false
}

func isYouTube(args []string) bool {
	for _, a := range args {
		l := strings.ToLower(a)
		if strings.Contains(l, "youtube.com/") || strings.Contains(l, "youtu.be/") {
			return true
		}
	}
	return false
}

func passthrough(real, deno string, args []string) int {
	cmd := exec.Command(real, args...)
	cmd.Env = childEnv(deno)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	err := cmd.Run()
	if err == nil {
		return 0
	}
	if ee, ok := err.(*exec.ExitError); ok {
		return ee.ExitCode()
	}
	fmt.Fprintln(os.Stderr, err)
	return 1
}

func cleanArgs(args []string) []string {
	out := make([]string, 0, len(args)+4)
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "-f" || a == "--format" {
			if i+1 < len(args) {
				i++
			}
			continue
		}
		if strings.HasPrefix(a, "--format=") {
			continue
		}
		if strings.HasPrefix(a, "-f") && len(a) > 2 {
			continue
		}
		if a == "--extractor-args" {
			if i+1 < len(args) {
				i++
			}
			continue
		}
		if strings.HasPrefix(a, "--extractor-args=") {
			continue
		}
		if a == "--js-runtimes" {
			if i+1 < len(args) {
				i++
			}
			continue
		}
		if strings.HasPrefix(a, "--js-runtimes=") {
			continue
		}
		if a == "--check-formats" || a == "--no-check-formats" || a == "--check-all-formats" {
			continue
		}
		out = append(out, a)
	}
	return out
}

func extractURLs(stdout string) []string {
	var urls []string
	for _, line := range strings.Split(stdout, "\n") {
		s := strings.TrimSpace(line)
		if strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") {
			urls = append(urls, s)
		}
	}
	return urls
}

func summarizeURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return "url=parse_error"
	}
	q := u.Query()
	itag := q.Get("itag")
	client := q.Get("c")
	if itag == "" {
		parts := strings.Split(u.Path, "/")
		for i := 0; i+1 < len(parts); i++ {
			if parts[i] == "itag" {
				itag = parts[i+1]
				break
			}
		}
	}
	if client == "" {
		client = "-"
	}
	if itag == "" {
		itag = "-"
	}
	return fmt.Sprintf("host=%s itag=%s client=%s", u.Hostname(), itag, client)
}

func preflight(raw string) (int, string, error) {
	client := &http.Client{Timeout: 8 * time.Second, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 8 {
			return fmt.Errorf("too many redirects")
		}
		return nil
	}}
	req, err := http.NewRequest("GET", raw, nil)
	if err != nil {
		return 0, "", err
	}
	req.Header.Set("Range", "bytes=0-1023")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Encoding", "identity")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/151.0.0.0 Safari/537.36")
	resp, err := client.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()
	_, _ = io.CopyN(io.Discard, resp.Body, 1024)
	return resp.StatusCode, resp.Header.Get("Content-Type"), nil
}

func runAttempt(real, deno string, base []string, a attempt) (string, string, int, error) {
	args := make([]string, 0, len(base)+10)
	args = append(args, "--js-runtimes", "deno")
	if a.extractorArgs != "" {
		args = append(args, "--extractor-args", a.extractorArgs)
	}
	args = append(args, "--check-formats", "-f", formatSelector)
	args = append(args, base...)
	var stdout, stderr bytes.Buffer
	cmd := exec.Command(real, args...)
	cmd.Env = childEnv(deno)
	cmd.Stdout, cmd.Stderr, cmd.Stdin = &stdout, &stderr, os.Stdin
	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			exitCode = ee.ExitCode()
		} else {
			exitCode = 1
		}
	}
	return stdout.String(), stderr.String(), exitCode, err
}

func main() {
	exePath, err := os.Executable()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	exeDir := filepath.Dir(exePath)
	logDir := filepath.Dir(cacheDir())
	_ = os.MkdirAll(logDir, 0755)
	real, deno, err := ensureDeps(exeDir, logDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "VRChat yt-dlp one-file: dependency setup failed:", err)
		logLine(logDir, "BOOTSTRAP ERROR "+err.Error())
		os.Exit(1)
	}

	incoming := os.Args[1:]
	if !hasHTTPURL(incoming) || !isYouTube(incoming) {
		os.Exit(passthrough(real, deno, incoming))
	}

	base := cleanArgs(incoming)
	logLine(logDir, "=== onefile v1 invocation ===")
	logLine(logDir, "IN : "+quoteArgs(incoming))
	logLine(logDir, "BASE: "+quoteArgs(base))
	logLine(logDir, "FORMAT: "+formatSelector)

	lastExit, lastErrText := 1, ""
	for _, a := range attempts {
		logLine(logDir, "TRY "+a.name+" extractor_args="+a.extractorArgs)
		stdout, stderr, exitCode, runErr := runAttempt(real, deno, base, a)
		urls := extractURLs(stdout)
		logLine(logDir, fmt.Sprintf("RESULT %s exit=%d urls=%d", a.name, exitCode, len(urls)))
		if stderr != "" {
			logLine(logDir, "STDERR "+a.name+": "+strings.ReplaceAll(strings.TrimSpace(stderr), "\r", ""))
		}
		if runErr == nil && exitCode == 0 && len(urls) > 0 {
			candidate := urls[len(urls)-1]
			status, ctype, pfErr := preflight(candidate)
			if pfErr != nil {
				logLine(logDir, "PREFLIGHT "+a.name+" "+summarizeURL(candidate)+" error="+pfErr.Error())
				lastErrText = stderr
				continue
			}
			logLine(logDir, fmt.Sprintf("PREFLIGHT %s %s status=%d content_type=%s", a.name, summarizeURL(candidate), status, ctype))
			if status >= 200 && status < 300 {
				fmt.Println(candidate)
				if stderr != "" {
					fmt.Fprint(os.Stderr, stderr)
				}
				logLine(logDir, "SUCCESS "+a.name+" validated")
				return
			}
			logLine(logDir, fmt.Sprintf("REJECT %s HTTP=%d", a.name, status))
			lastErrText = stderr
			continue
		}
		lastExit, lastErrText = exitCode, stderr
	}
	if lastErrText != "" {
		fmt.Fprint(os.Stderr, lastErrText)
	} else {
		fmt.Fprintln(os.Stderr, "VRChat yt-dlp one-file: no validated combined audio/video URL was available")
	}
	logLine(logDir, "FAILED all attempts")
	if lastExit == 0 {
		lastExit = 1
	}
	os.Exit(lastExit)
}
