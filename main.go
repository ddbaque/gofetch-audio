// Package main provides the CLI entry point for gofetch-audio,
// a tool to download audio from YouTube videos.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ddbaque/gofetch-audio/internal/downloader"
	"github.com/ddbaque/gofetch-audio/internal/tui"
)

type Config struct {
	URLs      []string
	OutputDir string
	Format    string
	Quality   string
	Parallel  int
}

func main() {
	config := parseFlags()

	if len(config.URLs) == 0 {
		fmt.Fprintln(os.Stderr, "Error: at least one URL is required")
		flag.Usage()
		os.Exit(1)
	}

	if err := downloader.CheckDependencies(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if err := os.MkdirAll(config.OutputDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output directory: %v\n", err)
		os.Exit(1)
	}

	dlConfig := downloader.Config{
		OutputDir: config.OutputDir,
		Format:    config.Format,
		Quality:   config.Quality,
	}

	model := tui.NewModel(config.URLs, dlConfig, config.Parallel)
	p := tea.NewProgram(model)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func parseFlags() Config {
	var config Config
	var urlList string
	var urlFile string

	flag.StringVar(&urlList, "urls", "", "Comma-separated list of YouTube URLs")
	flag.StringVar(&urlFile, "file", "", "File containing URLs (one per line)")
	flag.StringVar(&config.OutputDir, "output", ".", "Output directory for downloaded audio")
	flag.StringVar(&config.Format, "format", "mp3", "Audio format (mp3, m4a, opus, wav)")
	flag.StringVar(&config.Quality, "quality", "192", "Audio quality in kbps (128, 192, 256, 320)")
	flag.IntVar(&config.Parallel, "parallel", 3, "Number of parallel downloads")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "gofetch-audio - Download audio from YouTube videos\n\n")
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  %s [options] <url1> [url2] [url3] ...\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s [options] -file urls.txt\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s [options] -urls \"url1,url2,url3\"\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s \"https://youtube.com/watch?v=xxx\"\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -file playlist.txt -output ./music\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -format m4a -quality 320 \"url1\" \"url2\"\n", os.Args[0])
	}

	flag.Parse()

	// Read URLs from file if specified
	if urlFile != "" {
		fileURLs, err := readURLsFromFile(urlFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading URL file: %v\n", err)
			os.Exit(1)
		}
		config.URLs = append(config.URLs, fileURLs...)
	}

	// Collect URLs from -urls flag
	if urlList != "" {
		config.URLs = append(config.URLs, strings.Split(urlList, ",")...)
	}

	// Collect URLs from positional arguments
	config.URLs = append(config.URLs, flag.Args()...)

	// Trim whitespace and filter empty URLs
	filtered := config.URLs[:0]
	for _, url := range config.URLs {
		url = strings.TrimSpace(url)
		if url != "" && !strings.HasPrefix(url, "#") {
			filtered = append(filtered, url)
		}
	}
	config.URLs = filtered

	// Convert output to absolute path
	if absPath, err := filepath.Abs(config.OutputDir); err == nil {
		config.OutputDir = absPath
	}

	return config
}

func readURLsFromFile(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var urls []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			urls = append(urls, line)
		}
	}

	return urls, scanner.Err()
}
