package ui

import (
	"context"
	"fmt"
	"html"
	"math"
	"os"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/diamondburned/gotk4/pkg/cairo"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"

	"github.com/abunjevac/devtabs/internal/config"
	"github.com/abunjevac/devtabs/internal/vte"
)

type tabState int

const (
	stateIdle    tabState = iota
	stateDelay            // waiting for startup_delay to elapse
	stateRunning          // foreground process group ≠ shell PID
	stateDead             // shell exited; terminal no longer usable
)

// tab owns a single VTE terminal page and its state machine.
type tab struct {
	cfg      *config.TabConfig
	terminal *vte.Terminal
	widget   gtk.Widgetter
	dot      *gtk.DrawingArea

	cancel context.CancelFunc

	mu            sync.Mutex
	state         tabState
	shellPID      int
	ptyFd         uintptr
	onStateChange func(tabState) // called on GTK main thread after every state change
}

func newTab(cfg *config.TabConfig) *tab {
	return &tab{cfg: cfg}
}

// focus moves keyboard focus to the VTE terminal widget.
func (t *tab) focus() {
	if w, ok := t.widget.(*gtk.Widget); ok {
		w.GrabFocus()
	}
}

// labelWidget returns the tab label. idx is the zero-based tab position;
// pass -1 to suppress the Alt+N shortcut hint.
func (t *tab) labelWidget(idx int) gtk.Widgetter {
	dot := gtk.NewDrawingArea()

	dot.SetSizeRequest(10, 10)
	dot.SetDrawFunc(t.drawDot)

	t.dot = dot

	name := gtk.NewLabel("")

	if idx >= 0 && idx < 9 {
		name.SetMarkup(fmt.Sprintf(
			"%s <span size='small' alpha='50%%'>alt+%d</span>",
			html.EscapeString(t.cfg.Name), idx+1,
		))
	} else {
		name.SetText(t.cfg.Name)
	}

	box := gtk.NewBox(gtk.OrientationHorizontal, 5)

	box.Append(dot)
	box.Append(name)
	box.SetVAlign(gtk.AlignCenter)

	return box
}

func (t *tab) drawDot(da *gtk.DrawingArea, cr *cairo.Context, width, height int) {
	t.mu.Lock()
	s := t.state
	t.mu.Unlock()

	cx := float64(width) / 2
	cy := float64(height) / 2
	r := cx - 1

	cr.Arc(cx, cy, r, 0, 2*math.Pi)

	switch s {
	case stateRunning:
		cr.SetSourceRGB(0.2, 0.75, 0.2) // green
	case stateDelay:
		cr.SetSourceRGB(1.0, 0.78, 0.0) // yellow
	case stateDead:
		cr.SetSourceRGB(0.75, 0.2, 0.2) // red
	case stateIdle:
		cr.SetSourceRGB(0.55, 0.55, 0.55) // grey
	}

	cr.Fill()
}

// setState sets the tab state and queues a dot redraw.
// Must be called from the GTK main thread.
func (t *tab) setState(s tabState) {
	t.mu.Lock()
	t.state = s
	t.mu.Unlock()

	if t.dot != nil {
		t.dot.QueueDraw()
	}

	if t.onStateChange != nil {
		t.onStateChange(s)
	}
}

// getState returns the current state.
func (t *tab) getState() tabState {
	t.mu.Lock()
	defer t.mu.Unlock()

	return t.state
}

// runCommand writes the configured command to the terminal PTY.
// Must be called from the GTK main thread.
func (t *tab) runCommand() {
	t.setState(stateRunning)

	vte.FeedChild(t.terminal, t.cfg.Command+"\n")
}

// scheduleStartup initiates run_on_startup behaviour after spawn completes.
// Must be called from the GTK main thread.
func (t *tab) scheduleStartup() {
	if !t.cfg.RunOnStartup {
		t.setState(stateIdle)

		return
	}

	delay := t.cfg.StartupDelay.Duration

	if delay == 0 {
		t.runCommand()

		return
	}

	t.setState(stateDelay)

	time.AfterFunc(delay, func() {
		glib.IdleAdd(func() {
			if t.getState() == stateDelay {
				t.runCommand()
			}
		})
	})
}

// onSpawnDone is called on the GTK main thread by goVteSpawnDone.
func (t *tab) onSpawnDone(pid int, ptyFd uintptr, err error) {
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "devtabs: tab %q: spawn error: %v\n", t.cfg.Name, err)

		return
	}

	t.mu.Lock()
	t.shellPID = pid
	t.ptyFd = ptyFd
	t.mu.Unlock()

	ctx, cancel := context.WithCancel(context.Background())

	t.cancel = cancel

	go t.pollState(ctx)

	t.scheduleStartup()
}

// die stops the poller and marks the terminal permanently dead after the shell exits.
// Must be called from the GTK main thread.
func (t *tab) die() {
	if t.cancel != nil {
		t.cancel()
	}

	t.setState(stateDead)
}

// close cancels the poller goroutine and sends SIGHUP to the shell.
func (t *tab) close() {
	if t.cancel != nil {
		t.cancel()
	}

	t.mu.Lock()
	pid := t.shellPID
	t.mu.Unlock()

	if pid > 0 {
		proc, err := os.FindProcess(pid)
		if err == nil {
			_ = proc.Signal(syscall.SIGHUP)
		}
	}
}

// pollState detects Running↔Idle transitions via TIOCGPGRP every 500ms.
// Runs in its own goroutine; all GTK mutations cross to the main thread via glib.IdleAdd.
func (t *tab) pollState(ctx context.Context) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			t.mu.Lock()
			pid := t.shellPID
			fd := t.ptyFd
			t.mu.Unlock()

			if pid == 0 || fd == 0 {
				continue
			}

			fg, err := foregroundPGID(fd)
			if err != nil {
				continue
			}

			isRunning := fg != pid

			glib.IdleAdd(func() {
				if isRunning {
					if t.getState() == stateIdle {
						t.setState(stateRunning)
					}
				} else {
					// poller only moves Running→Idle; never overrides Delay.
					if t.getState() == stateRunning {
						t.setState(stateIdle)
					}
				}
			})
		}
	}
}

func foregroundPGID(fd uintptr) (int, error) {
	var pgrp int32

	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, syscall.TIOCGPGRP, uintptr(unsafe.Pointer(&pgrp)))
	if errno != 0 {
		return -1, errno
	}

	return int(pgrp), nil
}
