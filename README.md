# gofetch-audio

A fast CLI tool to download audio from YouTube videos, built with Go and [Bubble Tea](https://github.com/charmbracelet/bubbletea).

## Features

- Download audio from multiple URLs in parallel
- Beautiful TUI with progress bars and spinners
- Support for MP3, M4A, OPUS, WAV formats
- Read URLs from file (one per line)
- Configurable audio quality (128-320 kbps)

## Requirements

### Linux / macOS

```bash
# yt-dlp
pipx install yt-dlp

# ffmpeg
apt install ffmpeg      # Debian/Ubuntu
brew install ffmpeg     # macOS
```

### Windows

```powershell
# Using winget
winget install yt-dlp
winget install ffmpeg

# Or using scoop
scoop install yt-dlp ffmpeg

# Or using chocolatey
choco install yt-dlp ffmpeg
```

> **Note:** Use [Windows Terminal](https://github.com/microsoft/terminal) for best experience with colors and emojis.

## Installation

```bash
git clone https://github.com/ddbaque/gofetch-audio.git
cd gofetch-audio
make build
```

Binary will be in `dist/gofetch-audio`.

### Cross-platform builds

```bash
make build-all
# Creates binaries for Linux, macOS (Intel/ARM), and Windows
```

## Usage

```bash
# Single URL
./gofetch-audio "https://youtube.com/watch?v=VIDEO_ID"

# Multiple URLs
./gofetch-audio "url1" "url2" "url3"

# From file
./gofetch-audio -file urls.txt

# Custom quality and output
./gofetch-audio -quality 320 -format m4a -output ./music "url"

# Parallel downloads
./gofetch-audio -parallel 5 -file playlist.txt
```

## Options

| Flag | Description | Default |
|------|-------------|---------|
| `-file` | File with URLs (one per line) | - |
| `-format` | Audio format (mp3, m4a, opus, wav) | mp3 |
| `-quality` | Audio quality in kbps | 192 |
| `-output` | Output directory | . |
| `-parallel` | Concurrent downloads | 3 |

## URL File Format

```
# Comments start with #
https://youtube.com/watch?v=xxx
https://youtube.com/watch?v=yyy
```

## License

MIT License - see [LICENSE](LICENSE)
