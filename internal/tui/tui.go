// Package tui provides a terminal user interface for gofetch-audio using
// Bubble Tea framework. It displays download progress with spinners, progress bars,
// and color-coded status indicators.
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ddbaque/gofetch-audio/internal/downloader"
)

var (
	// Colors
	primaryColor = lipgloss.Color("#7C3AED")
	successColor = lipgloss.Color("#10B981")
	errorColor   = lipgloss.Color("#EF4444")
	warningColor = lipgloss.Color("#F59E0B")
	mutedColor   = lipgloss.Color("#6B7280")

	// Styles
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(primaryColor).
			Padding(0, 1)

	_ = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(primaryColor).
		Padding(0, 1)

	successStyle = lipgloss.NewStyle().Foreground(successColor)
	errorStyle   = lipgloss.NewStyle().Foreground(errorColor)
	mutedStyle   = lipgloss.NewStyle().Foreground(mutedColor)
	warnStyle    = lipgloss.NewStyle().Foreground(warningColor)

	checkMark = successStyle.Render("✓")
	crossMark = errorStyle.Render("✗")
	pending   = mutedStyle.Render("○")
)

type Item struct {
	URL      string
	Title    string
	Status   downloader.Status
	Progress float64
	Error    error
}

type Model struct {
	items       []Item
	config      downloader.Config
	parallel    int
	spinner     spinner.Model
	progress    progress.Model
	progressCh  chan downloader.ProgressMsg
	activeCount int
	completed   int
	failed      int
	width       int
	quitting    bool
	done        bool
}

func NewModel(urls []string, config downloader.Config, parallel int) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(primaryColor)

	p := progress.New(
		progress.WithDefaultGradient(),
		progress.WithWidth(30),
		progress.WithoutPercentage(),
	)

	items := make([]Item, len(urls))
	for i, url := range urls {
		items[i] = Item{URL: url, Status: downloader.StatusPending}
	}

	return Model{
		items:      items,
		config:     config,
		parallel:   parallel,
		spinner:    s,
		progress:   p,
		progressCh: make(chan downloader.ProgressMsg, 100),
		width:      80,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.startDownloads(),
		m.waitForProgress(),
	)
}

func (m *Model) startDownloads() tea.Cmd {
	return func() tea.Msg {
		started := 0
		for i := range m.items {
			if started >= m.parallel {
				break
			}
			if m.items[i].Status == downloader.StatusPending {
				go downloader.Download(i, m.items[i].URL, m.config, m.progressCh)
				started++
				m.activeCount++
			}
		}
		return nil
	}
}

func (m Model) waitForProgress() tea.Cmd {
	return func() tea.Msg {
		return <-m.progressCh
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.progress.Width = min(msg.Width-20, 40)

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case downloader.ProgressMsg:
		return m.handleProgress(msg)
	}

	return m, nil
}

func (m Model) handleProgress(msg downloader.ProgressMsg) (tea.Model, tea.Cmd) {
	if msg.ID < 0 || msg.ID >= len(m.items) {
		return m, m.waitForProgress()
	}

	item := &m.items[msg.ID]
	item.Status = msg.Status
	item.Progress = msg.Progress
	if msg.Title != "" {
		item.Title = msg.Title
	}
	if msg.Error != nil {
		item.Error = msg.Error
	}

	var cmds []tea.Cmd
	cmds = append(cmds, m.waitForProgress())

	// Handle completion
	if msg.Status == downloader.StatusCompleted || msg.Status == downloader.StatusFailed {
		m.activeCount--
		if msg.Status == downloader.StatusCompleted {
			m.completed++
		} else {
			m.failed++
		}

		// Start next download if any pending
		for i := range m.items {
			if m.items[i].Status == downloader.StatusPending && m.activeCount < m.parallel {
				go downloader.Download(i, m.items[i].URL, m.config, m.progressCh)
				m.activeCount++
				break
			}
		}

		// Check if all done
		if m.completed+m.failed == len(m.items) {
			m.done = true
			return m, tea.Quit
		}
	}

	cmds = append(cmds, m.spinner.Tick)
	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	if m.quitting {
		return "\n  Cancelled.\n\n"
	}

	var b strings.Builder

	// Header
	header := titleStyle.Render(" gofetch-audio ")
	stats := fmt.Sprintf(" %d/%d ", m.completed+m.failed, len(m.items))
	statsStyled := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(mutedColor).
		Render(stats)

	b.WriteString("\n  " + header + " " + statsStyled + "\n\n")

	// Items
	maxItems := 10 // Show max 10 items at a time
	start := 0
	if len(m.items) > maxItems {
		// Find first non-completed item
		for i, item := range m.items {
			if item.Status != downloader.StatusCompleted && item.Status != downloader.StatusFailed {
				start = max(0, i-2)
				break
			}
		}
		if start+maxItems > len(m.items) {
			start = max(0, len(m.items)-maxItems)
		}
	}

	end := min(start+maxItems, len(m.items))

	for i := start; i < end; i++ {
		item := m.items[i]
		b.WriteString(m.renderItem(item))
	}

	if len(m.items) > maxItems {
		hidden := len(m.items) - maxItems
		b.WriteString(mutedStyle.Render(fmt.Sprintf("  ... and %d more\n", hidden)))
	}

	// Footer
	b.WriteString("\n")
	footer := fmt.Sprintf("  %s %d completed  %s %d failed  %s %d pending",
		checkMark, m.completed,
		crossMark, m.failed,
		pending, len(m.items)-m.completed-m.failed)
	b.WriteString(mutedStyle.Render(footer) + "\n")

	if !m.done {
		b.WriteString(mutedStyle.Render("\n  Press q to quit\n"))
	}
	b.WriteString("\n")

	return b.String()
}

func (m Model) renderItem(item Item) string {
	title := item.Title
	if title == "" {
		title = truncate(item.URL, 45)
	} else {
		title = truncate(title, 45)
	}

	var status string
	var line string

	switch item.Status {
	case downloader.StatusPending:
		status = pending
		line = fmt.Sprintf("  %s %s\n", status, mutedStyle.Render(title))

	case downloader.StatusDownloading:
		status = m.spinner.View()
		prog := m.progress.ViewAs(item.Progress / 100)
		pct := fmt.Sprintf("%3.0f%%", item.Progress)
		line = fmt.Sprintf("  %s %s %s %s\n", status, title, prog, pct)

	case downloader.StatusConverting:
		status = warnStyle.Render("⚙")
		line = fmt.Sprintf("  %s %s %s\n", status, title, warnStyle.Render("converting..."))

	case downloader.StatusCompleted:
		status = checkMark
		line = fmt.Sprintf("  %s %s\n", status, successStyle.Render(title))

	case downloader.StatusFailed:
		status = crossMark
		errMsg := "failed"
		if item.Error != nil {
			errMsg = item.Error.Error()
		}
		line = fmt.Sprintf("  %s %s %s\n", status, title, errorStyle.Render("("+errMsg+")"))
	}

	return line
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
