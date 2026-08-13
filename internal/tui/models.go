package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/entro314-labs/ember2go/internal/disk"
	"github.com/entro314-labs/ember2go/internal/wim"
)

type Screen int

const (
	DiskSelectionScreen Screen = iota
	EditionSelectionScreen
	ConfirmationScreen
	ProgressScreen
	CompletionScreen
)

type Model struct {
	currentScreen   Screen
	disks           []*disk.Disk
	editions        []*wim.WindowsEdition
	selectedDisk    int
	selectedEdition int
	isoPath         string
	confirmed       bool
	progress        ProgressModel
	err             error
	quitting        bool
}

type ProgressModel struct {
	current    int
	total      int
	stage      string
	startTime  time.Time
	lastUpdate time.Time
	speed      float64
	eta        time.Duration
}

func NewModel(isoPath string) Model {
	return Model{
		currentScreen:   DiskSelectionScreen,
		isoPath:         isoPath,
		selectedDisk:    0,
		selectedEdition: 0,
		progress: ProgressModel{
			startTime: time.Now(),
		},
	}
}

func (m Model) Init() tea.Cmd {
	return LoadDisks()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		case "up", "k":
			return m.handleUp()
		case "down", "j":
			return m.handleDown()
		case "enter":
			return m.handleEnter()
		case "esc":
			return m.handleEsc()
		}
	case DisksLoadedMsg:
		m.disks = msg.Disks
		if len(m.disks) == 0 {
			m.err = fmt.Errorf("no removable disks found")
		}
	case EditionsLoadedMsg:
		m.editions = msg.Editions
		if len(m.editions) == 0 {
			m.err = fmt.Errorf("no Windows editions found in ISO")
		} else {
			m.currentScreen = EditionSelectionScreen
		}
	case ProgressUpdateMsg:
		m.progress.current = msg.Current
		m.progress.total = msg.Total
		m.progress.stage = msg.Stage
		m.progress.lastUpdate = time.Now()

		elapsed := m.progress.lastUpdate.Sub(m.progress.startTime)
		if elapsed > 0 && m.progress.current > 0 {
			m.progress.speed = float64(m.progress.current) / elapsed.Seconds()
			if m.progress.speed > 0 {
				remaining := float64(m.progress.total - m.progress.current)
				m.progress.eta = time.Duration(remaining / m.progress.speed * float64(time.Second))
			}
		}
	case CompletionMsg:
		m.currentScreen = CompletionScreen
	case ErrorMsg:
		m.err = msg.Error
	}

	return m, nil
}

func (m Model) handleUp() (tea.Model, tea.Cmd) {
	switch m.currentScreen {
	case DiskSelectionScreen:
		if m.selectedDisk > 0 {
			m.selectedDisk--
		}
	case EditionSelectionScreen:
		if m.selectedEdition > 0 {
			m.selectedEdition--
		}
	}
	return m, nil
}

func (m Model) handleDown() (tea.Model, tea.Cmd) {
	switch m.currentScreen {
	case DiskSelectionScreen:
		if m.selectedDisk < len(m.disks)-1 {
			m.selectedDisk++
		}
	case EditionSelectionScreen:
		if m.selectedEdition < len(m.editions)-1 {
			m.selectedEdition++
		}
	}
	return m, nil
}

func (m Model) handleEnter() (tea.Model, tea.Cmd) {
	switch m.currentScreen {
	case DiskSelectionScreen:
		if len(m.disks) > 0 {
			return m, LoadEditions(m.isoPath)
		}
	case EditionSelectionScreen:
		m.currentScreen = ConfirmationScreen
	case ConfirmationScreen:
		if !m.confirmed {
			m.confirmed = true
			m.currentScreen = ProgressScreen
			m.progress.startTime = time.Now()
			return m, StartCreation(m.disks[m.selectedDisk], m.editions[m.selectedEdition], m.isoPath)
		}
	}
	return m, nil
}

func (m Model) handleEsc() (tea.Model, tea.Cmd) {
	switch m.currentScreen {
	case EditionSelectionScreen:
		m.currentScreen = DiskSelectionScreen
	case ConfirmationScreen:
		m.currentScreen = EditionSelectionScreen
	default:
		m.quitting = true
		return m, tea.Quit
	}
	return m, nil
}

func (m Model) View() string {
	if m.quitting {
		return "Goodbye!\n"
	}

	if m.err != nil {
		return errorStyle.Render(fmt.Sprintf("Error: %v\n\nPress 'q' to quit.", m.err))
	}

	switch m.currentScreen {
	case DiskSelectionScreen:
		return m.diskSelectionView()
	case EditionSelectionScreen:
		return m.editionSelectionView()
	case ConfirmationScreen:
		return m.confirmationView()
	case ProgressScreen:
		return m.progressView()
	case CompletionScreen:
		return m.completionView()
	}

	return ""
}

func (m Model) diskSelectionView() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("📱 Select USB Drive"))
	b.WriteString("\n\n")

	if len(m.disks) == 0 {
		b.WriteString("🔍 Scanning for removable drives...\n")
		return b.String()
	}

	for i, disk := range m.disks {
		cursor := " "
		if i == m.selectedDisk {
			cursor = ">"
		}

		style := itemStyle
		if i == m.selectedDisk {
			style = selectedItemStyle
		}

		b.WriteString(style.Render(fmt.Sprintf("%s %s", cursor, disk.String())))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("↑/↓: navigate • enter: select • q: quit"))

	return b.String()
}

func (m Model) editionSelectionView() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("🪟 Select Windows Edition"))
	b.WriteString("\n\n")

	if len(m.editions) == 0 {
		b.WriteString("🔍 Analyzing Windows ISO...\n")
		return b.String()
	}

	for i, edition := range m.editions {
		cursor := " "
		if i == m.selectedEdition {
			cursor = ">"
		}

		style := itemStyle
		if i == m.selectedEdition {
			style = selectedItemStyle
		}

		b.WriteString(style.Render(fmt.Sprintf("%s %d: %s", cursor, i+1, edition.DisplayName)))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("↑/↓: navigate • enter: select • esc: back • q: quit"))

	return b.String()
}

func (m Model) confirmationView() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("⚠️  Confirmation"))
	b.WriteString("\n\n")

	b.WriteString("You are about to:\n")
	b.WriteString(fmt.Sprintf("• Erase all data on %s\n", m.disks[m.selectedDisk].String()))
	b.WriteString(fmt.Sprintf("• Install %s\n", m.editions[m.selectedEdition].DisplayName))
	b.WriteString(fmt.Sprintf("• From ISO: %s\n", m.isoPath))
	b.WriteString("\n")

	b.WriteString(warningStyle.Render("⚠️  THIS WILL PERMANENTLY DELETE ALL DATA ON THE SELECTED DRIVE!"))
	b.WriteString("\n\n")

	if !m.confirmed {
		b.WriteString(dangerStyle.Render("Press ENTER to continue or ESC to cancel"))
	} else {
		b.WriteString("Starting creation process...\n")
	}

	return b.String()
}

func (m Model) progressView() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("🚀 Creating Windows To Go"))
	b.WriteString("\n\n")

	if m.progress.stage != "" {
		b.WriteString(fmt.Sprintf("Current: %s\n", m.progress.stage))
		b.WriteString("\n")
	}

	if m.progress.total > 0 {
		percent := float64(m.progress.current) / float64(m.progress.total) * 100
		progressBar := m.renderProgressBar(percent)
		b.WriteString(progressBar)
		b.WriteString("\n")
		b.WriteString(fmt.Sprintf("Progress: %d/%d (%.1f%%)\n", m.progress.current, m.progress.total, percent))

		if m.progress.speed > 0 {
			b.WriteString(fmt.Sprintf("Speed: %.1f MB/s\n", m.progress.speed/1024/1024))
		}

		if m.progress.eta > 0 {
			b.WriteString(fmt.Sprintf("ETA: %s\n", m.progress.eta.Round(time.Second)))
		}
	} else {
		b.WriteString("Preparing...\n")
	}

	elapsed := time.Since(m.progress.startTime)
	b.WriteString(fmt.Sprintf("Elapsed: %s\n", elapsed.Round(time.Second)))

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("ctrl+c: cancel"))

	return b.String()
}

func (m Model) renderProgressBar(percent float64) string {
	width := 50
	filled := int(percent * float64(width) / 100)

	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	return progressBarStyle.Render(bar)
}

func (m Model) completionView() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("✅ Windows To Go Created Successfully!"))
	b.WriteString("\n\n")

	b.WriteString("Your Windows To Go drive is ready!\n\n")

	b.WriteString("Next steps:\n")
	b.WriteString("1. Safely eject the USB drive\n")
	b.WriteString("2. Boot from USB in UEFI mode\n")
	b.WriteString("3. Windows should start automatically\n\n")

	elapsed := time.Since(m.progress.startTime)
	b.WriteString(fmt.Sprintf("Total time: %s\n", elapsed.Round(time.Second)))

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("Press any key to exit"))

	return b.String()
}

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1)

	itemStyle = lipgloss.NewStyle().
			PaddingLeft(2)

	selectedItemStyle = lipgloss.NewStyle().
				PaddingLeft(2).
				Foreground(lipgloss.Color("#7D56F4")).
				Bold(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#626262"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF5F87")).
			Bold(true)

	warningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFAA00")).
			Bold(true)

	dangerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF0000")).
			Bold(true).
			Blink(true)

	progressBarStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#7D56F4"))
)
