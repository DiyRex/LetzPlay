// Package tui is the interactive terminal UI shown when the desktop binary runs in the
// foreground. It lets the host pick the audio output device, watch what's playing live, see who's
// connected, and control playback — while the web server runs in the background.
//
// Built on Bubble Tea (the Elm-style model/update/view pattern), which keeps all state in one
// immutable model and all mutation in Update, avoiding the tangled callback state that ad-hoc
// TUIs accumulate.
package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/DiyRex/LetzPlay/desktop/internal/api"
	"github.com/DiyRex/LetzPlay/desktop/internal/domain"
	"github.com/DiyRex/LetzPlay/desktop/internal/player"
)

var (
	accent    = lipgloss.Color("#7C5CFF")
	muted     = lipgloss.Color("#A0A0B0")
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(accent)
	labelStyle = lipgloss.NewStyle().Foreground(muted)
	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#3B2F73")).
			Padding(0, 1).
			MarginBottom(1)
	selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Bold(true)
	helpStyle     = lipgloss.NewStyle().Foreground(muted)
)

// Model is the full TUI state.
type Model struct {
	ctx     context.Context
	queue   *domain.Queue
	player  *player.Mpv
	hub     *api.Hub
	url     string

	updates <-chan domain.Snapshot
	snap    domain.Snapshot
	devices []player.AudioDevice
	users   []api.ConnectedUser
	cursor  int
	width   int
}

// New builds a Model and subscribes it to the queue's live snapshots.
func New(ctx context.Context, q *domain.Queue, p *player.Mpv, hub *api.Hub, url string) Model {
	updates, _ := q.Subscribe() // lifetime == program; unsubscribe happens on process exit
	return Model{ctx: ctx, queue: q, player: p, hub: hub, url: url, updates: updates, snap: q.Snapshot()}
}

// --- messages ---

type snapshotMsg domain.Snapshot
type devicesMsg []player.AudioDevice
type tickMsg struct{}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.waitForSnapshot(), m.loadDevices(), tick())
}

func (m Model) waitForSnapshot() tea.Cmd {
	return func() tea.Msg { return snapshotMsg(<-m.updates) }
}

func (m Model) loadDevices() tea.Cmd {
	return func() tea.Msg {
		devices, err := m.player.AudioDevices(m.ctx)
		if err != nil {
			return devicesMsg(nil)
		}
		return devicesMsg(devices)
	}
}

func tick() tea.Cmd {
	return tea.Tick(2*time.Second, func(time.Time) tea.Msg { return tickMsg{} })
}

// --- update ---

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		return m, nil

	case snapshotMsg:
		m.snap = domain.Snapshot(msg)
		return m, m.waitForSnapshot() // keep listening

	case devicesMsg:
		m.devices = []player.AudioDevice(msg)
		if m.cursor >= len(m.devices) {
			m.cursor = 0
		}
		return m, nil

	case tickMsg:
		m.users = m.hub.Users()
		return m, tick()

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.devices)-1 {
			m.cursor++
		}
	case "enter":
		if m.cursor < len(m.devices) {
			m.player.SetAudioDevice(m.devices[m.cursor].Name)
		}
	case " ":
		if m.snap.Status == domain.StatusPlaying || m.snap.Status == domain.StatusBuffering {
			m.player.Pause()
		} else {
			m.player.Play()
		}
	case "n", "right":
		m.queue.Advance()
	case "+", "=":
		m.setVolume(m.snap.Volume + 5)
	case "-", "_":
		m.setVolume(m.snap.Volume - 5)
	case "r":
		return m, m.loadDevices()
	}
	return m, nil
}

func (m *Model) setVolume(v int) {
	if v < 0 {
		v = 0
	}
	if v > 100 {
		v = 100
	}
	m.player.SetVolume(v)
	m.queue.SetVolume(v)
}

// --- view ---

func (m Model) View() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("♫ LetzPlay Musix") + labelStyle.Render("   "+m.url) + "\n\n")
	b.WriteString(m.renderNowPlaying())
	b.WriteString(m.renderDevices())
	b.WriteString(m.renderPresence())
	b.WriteString(helpStyle.Render(
		"↑/↓ device · enter select · space play/pause · n skip · +/- volume · r refresh · q quit"))
	return b.String()
}

func (m Model) renderNowPlaying() string {
	var body strings.Builder
	song := m.snap.Current()
	if song == nil {
		body.WriteString(labelStyle.Render("Nothing playing — add a song from the web remote."))
	} else {
		upNext := 0
		if m.snap.CurrentIndex >= 0 {
			upNext = len(m.snap.Tracks) - m.snap.CurrentIndex - 1
		}
		body.WriteString(selectedStyle.Render(truncate(song.Title, 56)) + "\n")
		body.WriteString(labelStyle.Render(fmt.Sprintf("added by %s · %s", song.AddedBy, m.snap.Status)) + "\n")
		body.WriteString(renderProgress(m.snap.PositionSeconds, m.snap.DurationSeconds, 40) + "\n")
		body.WriteString(labelStyle.Render(fmt.Sprintf(
			"%s / %s   vol %d%%   %d in list · %d up next",
			formatTime(m.snap.PositionSeconds), formatTime(m.snap.DurationSeconds),
			m.snap.Volume, len(m.snap.Tracks), upNext)))
	}
	return panelStyle.Render(labelStyle.Render("NOW PLAYING")+"\n"+body.String()) + "\n"
}

func (m Model) renderDevices() string {
	var body strings.Builder
	if len(m.devices) == 0 {
		body.WriteString(labelStyle.Render("No audio devices reported (press r to refresh)."))
	} else {
		current := m.player.CurrentDevice()
		for i, d := range m.devices {
			marker := "  "
			if i == m.cursor {
				marker = "▶ "
			}
			line := fmt.Sprintf("%s%s", marker, truncate(displayName(d), 50))
			if d.Name == current {
				line += "  ✓"
			}
			if i == m.cursor {
				body.WriteString(selectedStyle.Render(line))
			} else {
				body.WriteString(line)
			}
			body.WriteString("\n")
		}
	}
	return panelStyle.Render(labelStyle.Render("AUDIO OUTPUT")+"\n"+strings.TrimRight(body.String(), "\n")) + "\n"
}

func (m Model) renderPresence() string {
	if len(m.users) == 0 {
		return panelStyle.Render(labelStyle.Render("CONNECTED")+"\n"+labelStyle.Render("No remotes connected yet.")) + "\n"
	}
	names := make([]string, 0, len(m.users))
	for _, u := range m.users {
		tag := u.Username
		if u.Role.IsAdmin() {
			tag += " (admin)"
		}
		names = append(names, tag)
	}
	header := fmt.Sprintf("CONNECTED (%d)", len(m.users))
	return panelStyle.Render(labelStyle.Render(header)+"\n"+strings.Join(names, ", ")) + "\n"
}

func displayName(d player.AudioDevice) string {
	if d.Description != "" {
		return d.Description
	}
	return d.Name
}

func renderProgress(pos, dur float64, width int) string {
	filled := 0
	if dur > 0 {
		filled = int(pos / dur * float64(width))
		if filled > width {
			filled = width
		}
	}
	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	return lipgloss.NewStyle().Foreground(accent).Render(bar)
}

func formatTime(seconds float64) string {
	s := int(seconds)
	if s < 0 {
		s = 0
	}
	return fmt.Sprintf("%d:%02d", s/60, s%60)
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}
