// Package downloader provides functionality to download audio from YouTube videos
// using yt-dlp as the backend. It handles progress reporting and audio extraction.
package downloader

import (
	"bufio"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

type Status int

const (
	StatusPending Status = iota
	StatusDownloading
	StatusConverting
	StatusCompleted
	StatusFailed
)

type ProgressMsg struct {
	ID       int
	Status   Status
	Progress float64
	Title    string
	Error    error
}

type Config struct {
	OutputDir string
	Format    string
	Quality   string
}

func Download(id int, url string, config Config, progressChan chan<- ProgressMsg) {
	progressChan <- ProgressMsg{ID: id, Status: StatusDownloading, Progress: 0}

	args := []string{
		"--extract-audio",
		"--audio-format", config.Format,
		"--audio-quality", config.Quality + "K",
		"--output", filepath.Join(config.OutputDir, "%(title)s.%(ext)s"),
		"--no-playlist",
		"--no-overwrites",
		"--restrict-filenames",
		"--newline",
		"--progress",
		url,
	}

	cmd := exec.Command("yt-dlp", args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		progressChan <- ProgressMsg{ID: id, Status: StatusFailed, Error: err}
		return
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		progressChan <- ProgressMsg{ID: id, Status: StatusFailed, Error: err}
		return
	}

	if err := cmd.Start(); err != nil {
		progressChan <- ProgressMsg{ID: id, Status: StatusFailed, Error: err}
		return
	}

	downloadRe := regexp.MustCompile(`\[download\]\s+(\d+\.?\d*)%`)
	destRe := regexp.MustCompile(`Destination:\s+.*/(.+)\.(webm|m4a|mp3|opus|wav)`)
	extractRe := regexp.MustCompile(`\[ExtractAudio\]`)

	var title string

	// Read stdout
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()

			if matches := destRe.FindStringSubmatch(line); len(matches) > 1 {
				title = strings.ReplaceAll(matches[1], "_", " ")
				progressChan <- ProgressMsg{ID: id, Status: StatusDownloading, Title: title}
			}

			if matches := downloadRe.FindStringSubmatch(line); len(matches) > 1 {
				var progress float64
				fmt.Sscanf(matches[1], "%f", &progress)
				progressChan <- ProgressMsg{ID: id, Status: StatusDownloading, Progress: progress, Title: title}
			}

			if extractRe.MatchString(line) {
				progressChan <- ProgressMsg{ID: id, Status: StatusConverting, Progress: 100, Title: title}
			}
		}
	}()

	// Read stderr for errors
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			// Check for title in stderr too
			if matches := destRe.FindStringSubmatch(line); len(matches) > 1 {
				title = strings.ReplaceAll(matches[1], "_", " ")
			}
			if matches := downloadRe.FindStringSubmatch(line); len(matches) > 1 {
				var progress float64
				fmt.Sscanf(matches[1], "%f", &progress)
				progressChan <- ProgressMsg{ID: id, Status: StatusDownloading, Progress: progress, Title: title}
			}
		}
	}()

	if err := cmd.Wait(); err != nil {
		progressChan <- ProgressMsg{ID: id, Status: StatusFailed, Title: title, Error: fmt.Errorf("download failed")}
		return
	}

	progressChan <- ProgressMsg{ID: id, Status: StatusCompleted, Progress: 100, Title: title}
}

func CheckDependencies() error {
	if _, err := exec.LookPath("yt-dlp"); err != nil {
		return fmt.Errorf("yt-dlp not found. Install with: pipx install yt-dlp")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return fmt.Errorf("ffmpeg not found. Install with your package manager")
	}
	return nil
}
