package vte

// #cgo pkg-config: vte-2.91-gtk4
// #include "vte.h"
// #include <stdlib.h>
import "C"

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"unsafe"

	coreglib "github.com/diamondburned/gotk4/pkg/core/glib"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

// SpawnCallback is called on the GTK main thread after vte_terminal_spawn_async completes.
type SpawnCallback func(pid int, ptyFd uintptr, err error)

// ChildExitedCallback is called on the GTK main thread when the shell exits.
type ChildExitedCallback func(status int)

//nolint:gochecknoglobals
var (
	spawnMu       sync.Mutex
	nextSpawnID   atomic.Int64
	spawnRegistry = make(map[int]SpawnCallback)

	exitMu       sync.Mutex
	nextTabID    atomic.Int64
	exitRegistry = make(map[int]ChildExitedCallback)
)

// registerSpawnCallback stores cb and returns an ID to pass to SpawnAsync.
func registerSpawnCallback(cb SpawnCallback) int {
	id := int(nextSpawnID.Add(1))

	spawnMu.Lock()
	spawnRegistry[id] = cb
	spawnMu.Unlock()

	return id
}

// registerChildExitedCallback stores cb and returns a tabID to pass to ConnectChildExited.
func registerChildExitedCallback(cb ChildExitedCallback) int {
	id := int(nextTabID.Add(1))

	exitMu.Lock()
	exitRegistry[id] = cb
	exitMu.Unlock()

	return id
}

//export goVteSpawnDone
func goVteSpawnDone(callbackID C.int, pid C.int, ptyFd C.int, errMsg *C.char) {
	id := int(callbackID)

	spawnMu.Lock()
	cb, ok := spawnRegistry[id]
	delete(spawnRegistry, id)
	spawnMu.Unlock()

	if !ok {
		return
	}

	var goErr error

	if errMsg != nil {
		goErr = errors.New(C.GoString(errMsg))
	}

	cb(int(pid), uintptr(ptyFd), goErr)
}

//export goVteChildExited
func goVteChildExited(tabID C.int, status C.int) {
	id := int(tabID)

	exitMu.Lock()
	cb, ok := exitRegistry[id]
	exitMu.Unlock()

	if !ok {
		return
	}

	cb(int(status))
}

// Terminal is an opaque handle to a VteTerminal widget.
type Terminal struct {
	ptr *C.VteTerminal
}

// NewTerminal creates a new VteTerminal and returns it alongside a gtk.Widgetter
// suitable for embedding in a GtkNotebook page.
func NewTerminal() (*Terminal, gtk.Widgetter) {
	terminal := C.vte_terminal_new()
	ptr := (*C.VteTerminal)(unsafe.Pointer(terminal))
	t := &Terminal{ptr: ptr}

	obj := coreglib.Take(unsafe.Pointer(terminal))
	casted := obj.WalkCast(func(o coreglib.Objector) bool {
		_, ok := o.(gtk.Widgetter)

		return ok
	})

	widget, ok := casted.(gtk.Widgetter)
	if !ok {
		panic("vte: VteTerminal does not walk-cast to gtk.Widgetter")
	}

	return t, widget
}

// SpawnAsync starts a shell in the terminal asynchronously.
// cb is called on the GTK main thread with the shell PID and PTY fd.
func SpawnAsync(t *Terminal, workingDir, shell string, shellArgs []string, cb SpawnCallback) {
	id := registerSpawnCallback(cb)

	argv := buildArgv(shell, shellArgs)
	defer freeArgv(argv)

	var cwd *C.char

	if workingDir != "" {
		cwd = C.CString(workingDir)
		defer C.free(unsafe.Pointer(cwd)) //nolint:nlreturn // wtf?
	}

	C.vteSpawnAsync(t.ptr, cwd, &argv[0], C.int(id))
}

// ConnectChildExited wires the child-exited VTE signal.
func ConnectChildExited(t *Terminal, cb ChildExitedCallback) {
	id := registerChildExitedCallback(cb)

	C.vteConnectChildExited(t.ptr, C.int(id))
}

// SetFont applies a Pango font description to the terminal (e.g. family="Monospace", size=12).
func SetFont(t *Terminal, family string, size float64) {
	desc := fmt.Sprintf("%s %g", family, size)

	cs := C.CString(desc)
	defer C.free(unsafe.Pointer(cs)) //nolint:nlreturn // wtf?

	C.vteSetFont(t.ptr, cs)
}

// FeedChild writes data to the terminal PTY.
func FeedChild(t *Terminal, data string) {
	if len(data) == 0 {
		return
	}

	cs := C.CString(data)
	defer C.free(unsafe.Pointer(cs)) //nolint:nlreturn // wtf?

	C.vteFeedChild(t.ptr, cs, C.int(len(data)))
}

func buildArgv(shell string, args []string) []*C.char {
	argv := make([]*C.char, 0, len(args)+2)

	argv = append(argv, C.CString(shell))

	for _, a := range args {
		argv = append(argv, C.CString(a))
	}

	argv = append(argv, nil)

	return argv
}

func freeArgv(argv []*C.char) {
	for _, p := range argv {
		if p != nil {
			C.free(unsafe.Pointer(p))
		}
	}
}
