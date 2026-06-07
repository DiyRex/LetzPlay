// Package player drives audio playback on the desktop host via mpv.
//
// Why mpv: on a laptop wired to a speaker we want reliable, headless audio with no GUI toolkit
// dependency. mpv (with yt-dlp installed) resolves and plays a YouTube URL's audio directly and
// exposes a clean JSON IPC socket for control and events — which maps neatly onto our Player
// interface, lets us auto-advance the queue when a track ends, and lets the TUI enumerate and
// switch the audio output device live.
//
// Requires `mpv` and `yt-dlp` on PATH. See the repo README for install instructions.
package player

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/DiyRex/LetzPlay/desktop/internal/domain"
)

// AudioDevice is one selectable output, as reported by mpv's audio-device-list.
type AudioDevice struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// Mpv implements domain.Player by talking to an mpv subprocess over a unix IPC socket.
type Mpv struct {
	onStatus   func(domain.PlaybackStatus)
	onProgress func(position, duration float64)
	onEnded    func()

	socketPath string
	cmd        *exec.Cmd

	writeMu  sync.Mutex
	conn     net.Conn
	duration float64

	reqMu     sync.Mutex
	nextReqID int
	pending   map[int]chan ipcReply

	deviceMu sync.RWMutex
	current  string

	// Status is derived from three mpv signals so we report honestly: a track is only PLAYING
	// when a file is loaded, not paused, and actually decoding (not buffering). These are touched
	// only from the single readLoop goroutine.
	loaded   bool
	paused   bool
	coreIdle bool
}

// NewMpv wires the player to the queue via callbacks (status in, progress in, auto-advance on end).
func NewMpv(
	onStatus func(domain.PlaybackStatus),
	onProgress func(position, duration float64),
	onEnded func(),
) *Mpv {
	return &Mpv{
		onStatus:   onStatus,
		onProgress: onProgress,
		onEnded:    onEnded,
		nextReqID:  100, // keep clear of observe_property ids
		pending:    make(map[int]chan ipcReply),
	}
}

// Start launches mpv and connects to its IPC socket. Returns an error if mpv can't be started.
func (m *Mpv) Start(ctx context.Context) error {
	if _, err := exec.LookPath("mpv"); err != nil {
		return fmt.Errorf("mpv not found on PATH: %w", err)
	}
	m.socketPath = filepath.Join(os.TempDir(), fmt.Sprintf("letzplay-mpv-%d.sock", os.Getpid()))
	_ = os.Remove(m.socketPath)

	m.cmd = exec.CommandContext(ctx, "mpv",
		"--idle=yes",
		"--no-video",
		"--no-terminal",
		"--ytdl=yes",
		"--volume=100",
		// Network read-ahead so playback doesn't stall mid-song once it starts.
		"--cache=yes",
		"--cache-secs=120",
		"--demuxer-max-bytes=64MiB",
		"--input-ipc-server="+m.socketPath,
	)
	if err := m.cmd.Start(); err != nil {
		return fmt.Errorf("starting mpv: %w", err)
	}

	conn, err := m.dialWithRetry(ctx)
	if err != nil {
		return fmt.Errorf("connecting to mpv ipc: %w", err)
	}
	m.conn = conn

	go m.readLoop(conn)

	m.observe("time-pos", 1)
	m.observe("duration", 2)
	m.observe("pause", 3)
	m.observe("core-idle", 4) // true while not actively decoding (buffering/seeking/idle)
	return nil
}

// Stop terminates the mpv subprocess and removes its socket.
func (m *Mpv) Stop() {
	if m.conn != nil {
		_ = m.conn.Close()
	}
	if m.cmd != nil && m.cmd.Process != nil {
		_ = m.cmd.Process.Kill()
	}
	if m.socketPath != "" {
		_ = os.Remove(m.socketPath)
	}
}

// --- domain.Player ---

func (m *Mpv) Load(videoID string) {
	m.LoadURL("https://www.youtube.com/watch?v=" + videoID)
}

// LoadURL plays an already-resolved media URL directly. The prefetcher uses this with a yt-dlp
// pre-resolved direct stream URL so mpv skips the (slow) resolution step at transition time.
func (m *Mpv) LoadURL(url string) {
	m.command("loadfile", url, "replace")
}
func (m *Mpv) Play()                 { m.command("set_property", "pause", false) }
func (m *Mpv) Pause()                { m.command("set_property", "pause", true) }
func (m *Mpv) Seek(seconds float64)  { m.command("seek", seconds, "absolute") }
func (m *Mpv) SetVolume(percent int) { m.command("set_property", "volume", clampVolume(percent)) }

// SetLoop loops the current file forever (RepeatOne) or not. mpv emits no end-file while looping,
// so the queue never advances — exactly the repeat-one behaviour.
func (m *Mpv) SetLoop(loop bool) {
	if loop {
		m.command("set_property", "loop-file", "inf")
	} else {
		m.command("set_property", "loop-file", "no")
	}
}

// --- audio device control (used by the TUI) ---

// AudioDevices returns the host's available output devices, as mpv sees them.
func (m *Mpv) AudioDevices(ctx context.Context) ([]AudioDevice, error) {
	data, err := m.request(ctx, "get_property", "audio-device-list")
	if err != nil {
		return nil, err
	}
	var devices []AudioDevice
	if err := json.Unmarshal(data, &devices); err != nil {
		return nil, err
	}
	return devices, nil
}

// SetAudioDevice switches the output device live and remembers the selection.
func (m *Mpv) SetAudioDevice(name string) {
	m.command("set_property", "audio-device", name)
	m.deviceMu.Lock()
	m.current = name
	m.deviceMu.Unlock()
}

// CurrentDevice returns the last device set via SetAudioDevice ("" = mpv default/auto).
func (m *Mpv) CurrentDevice() string {
	m.deviceMu.RLock()
	defer m.deviceMu.RUnlock()
	return m.current
}

// --- IPC plumbing ---

func (m *Mpv) dialWithRetry(ctx context.Context) (net.Conn, error) {
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if conn, err := net.Dial("unix", m.socketPath); err == nil {
			return conn, nil
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(50 * time.Millisecond):
		}
	}
	return nil, errors.New("mpv ipc socket never appeared")
}

func (m *Mpv) observe(property string, id int) { m.command("observe_property", id, property) }

// command writes one fire-and-forget JSON command line.
func (m *Mpv) command(args ...any) {
	m.write(map[string]any{"command": args})
}

// request issues a command with a request_id and waits for the matching reply's data payload.
func (m *Mpv) request(ctx context.Context, args ...any) (json.RawMessage, error) {
	m.reqMu.Lock()
	id := m.nextReqID
	m.nextReqID++
	ch := make(chan ipcReply, 1)
	m.pending[id] = ch
	m.reqMu.Unlock()

	defer func() {
		m.reqMu.Lock()
		delete(m.pending, id)
		m.reqMu.Unlock()
	}()

	m.write(map[string]any{"command": args, "request_id": id})

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(3 * time.Second):
		return nil, errors.New("mpv request timed out")
	case reply := <-ch:
		if reply.Error != "" && reply.Error != "success" {
			return nil, fmt.Errorf("mpv error: %s", reply.Error)
		}
		return reply.Data, nil
	}
}

// write serializes and sends a message. Errors are swallowed: commands issued before mpv is
// ready are simply dropped, like the Android player.
func (m *Mpv) write(message map[string]any) {
	m.writeMu.Lock()
	defer m.writeMu.Unlock()
	if m.conn == nil {
		return
	}
	payload, err := json.Marshal(message)
	if err != nil {
		return
	}
	_, _ = m.conn.Write(append(payload, '\n'))
}

func (m *Mpv) readLoop(conn net.Conn) {
	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 0, 64*1024), 1<<20)
	for scanner.Scan() {
		var msg ipcMessage
		if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
			continue
		}
		if msg.RequestID != 0 {
			m.deliverReply(msg)
			continue
		}
		m.handleEvent(msg)
	}
}

type ipcMessage struct {
	Event     string          `json:"event"`
	Name      string          `json:"name"`
	Data      json.RawMessage `json:"data"`
	Reason    string          `json:"reason"`
	RequestID int             `json:"request_id"`
	Error     string          `json:"error"`
}

type ipcReply struct {
	Data  json.RawMessage
	Error string
}

func (m *Mpv) deliverReply(msg ipcMessage) {
	m.reqMu.Lock()
	ch, ok := m.pending[msg.RequestID]
	m.reqMu.Unlock()
	if ok {
		ch <- ipcReply{Data: msg.Data, Error: msg.Error}
	}
}

func (m *Mpv) handleEvent(msg ipcMessage) {
	switch msg.Event {
	case "start-file":
		m.loaded = true
		m.emitStatus()
	case "end-file":
		m.loaded = false
		// Advance only when a track finishes or errors — not on our own "replace" stop.
		if msg.Reason == "eof" || msg.Reason == "error" {
			m.onEnded()
		}
	case "property-change":
		m.handlePropertyChange(msg)
	}
}

func (m *Mpv) handlePropertyChange(msg ipcMessage) {
	switch msg.Name {
	case "time-pos":
		var pos float64
		if json.Unmarshal(msg.Data, &pos) == nil {
			m.onProgress(pos, m.duration)
		}
	case "duration":
		var dur float64
		if json.Unmarshal(msg.Data, &dur) == nil {
			m.duration = dur
		}
	case "pause":
		_ = json.Unmarshal(msg.Data, &m.paused)
		m.emitStatus()
	case "core-idle":
		_ = json.Unmarshal(msg.Data, &m.coreIdle)
		m.emitStatus()
	}
}

// emitStatus reports an honest status from the three mpv signals. Crucially it never claims
// PLAYING while the track is still buffering (core-idle) — which is what made a stalled song look
// like it was playing — and it stays quiet when nothing is loaded so the queue's IDLE state holds.
func (m *Mpv) emitStatus() {
	if !m.loaded {
		return
	}
	switch {
	case m.paused:
		m.onStatus(domain.StatusPaused)
	case m.coreIdle:
		m.onStatus(domain.StatusBuffering)
	default:
		m.onStatus(domain.StatusPlaying)
	}
}

func clampVolume(v int) int {
	if v < 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return v
}
