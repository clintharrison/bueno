package xkb

/*
// #cgo pkg-config: x11
#include <stdlib.h>
#include <X11/Xlib.h>
*/
import "C"
import (
	"fmt"
	"log/slog"
	"runtime"
	"unsafe"
)

func init() {
	// Xlib is not thread-safe by default, so we need to lock the OS thread
	// to ensure that all Xlib calls are made from the same thread.
	// TODO: figure out if XInitThreads can be used?
	runtime.LockOSThread()
}

type X11 struct {
	display    *C.Display
	rootWindow C.Window
}

//export onX11Error
func onX11Error(display *C.Display, error *C.XErrorEvent) {
	slog.Error("X11 error", "display", display, "error", error)
}

func Open() (*X11, error) {
	x := &X11{}
	name_cstr := C.XDisplayName(nil)
	if name_cstr == nil {
		return nil, fmt.Errorf("failed to get X display name")
	}
	name := C.GoString(name_cstr)
	slog.Info("opening X display", "name", name, "cstr", *name_cstr)

	x.display = C.XOpenDisplay(nil)
	if x.display == nil {
		return nil, fmt.Errorf("failed to open X display")
	}
	screen := C.XDefaultScreen(x.display)
	x.rootWindow = C.XRootWindow(x.display, screen)
	return x, nil
}

func (x *X11) Close() error {
	C.XCloseDisplay(x.display)
	return nil
}

type XKeysym uint32

const (
	XKPageUp   XKeysym = 0xFF55
	XKPageDown XKeysym = 0xFF56
)

func (x *X11) getActiveWindow() C.Window {
	atom := C.XInternAtom(x.display, C.CString("_NET_ACTIVE_WINDOW"), C.int(1))

	var actualType C.Atom
	var actualFormat C.int
	var nItems C.ulong
	var bytesAfter C.ulong
	var prop *C.uchar
	status := C.XGetWindowProperty(
		x.display,
		x.rootWindow,
		atom,
		0,        // long_offset
		1,        // long_length
		C.int(0), // delete
		C.AnyPropertyType,
		&actualType,
		&actualFormat,
		&nItems,
		&bytesAfter,
		&prop,
	)

	if status != C.Success || nItems == 0 || actualFormat == 0 {
		slog.Warn("failed to get active window", "status", status, "nItems", nItems, "actualFormat", actualFormat)
		return 0
	}
	propData := C.GoBytes(unsafe.Pointer(prop), C.int(nItems)*(actualFormat/8))
	if len(propData) != 4 {
		slog.Warn("expected 32-bit value in _NET_ACTIVE_WINDOW prop (i.e., actualFormat=32)", "length", len(propData), "actual_format", actualFormat)
		return 0
	}
	activeWindowId := uint32(propData[0]) | (uint32(propData[1]) << 8) | (uint32(propData[2]) << 16) | (uint32(propData[3]) << 24)
	// the data array will be leaked if not freed here
	C.free(unsafe.Pointer(prop))
	slog.Debug("got active window", "activeWindowId", activeWindowId, "actualType", actualType, "actualFormat", actualFormat, "nItems", nItems, "bytesAfter", bytesAfter)
	return C.Window(activeWindowId)
}

func (x *X11) KeyPress(keysym XKeysym) error {
	wnd := x.getActiveWindow()
	if wnd == 0 {
		slog.Warn("no active window, cannot send key event")
		return fmt.Errorf("no active window")
	}
	keycode := C.XKeysymToKeycode(x.display, C.KeySym(keysym))

	evt := C.XKeyEvent{
		display:   x.display,
		window:    wnd,
		subwindow: C.None,
		keycode:   C.uint(keycode),
		// TODO: handle state (e.g. modifier keys)? is that ever used for anything on kindle?
		state:       0,
		root:        x.rootWindow,
		same_screen: C.True,
		_type:       C.KeyPress,
		// xdotool doesn't know if these need to be set, and neither do I.
		// https://github.com/jordansissel/xdotool/blob/33092d8a74d60c9ad3ab39c4f05b90e047ea51d8/xdo.c#L1517-L1518
		x:      C.int(1),
		y:      C.int(1),
		x_root: C.int(1),
		y_root: C.int(1),
	}

	slog.Debug("calling XSendEvent", "keysym", keysym, "keycode", keycode, "type", "key_press", "wnd", wnd, "evt", evt)
	C.XSendEvent(x.display, wnd, C.True, C.KeyPressMask, (*C.XEvent)(unsafe.Pointer(&evt)))

	evt._type = C.KeyRelease
	slog.Debug("calling XSendEvent", "keysym", keysym, "keycode", keycode, "type", "key_release", "wnd", wnd, "evt", evt)
	C.XSendEvent(x.display, wnd, C.True, C.KeyPressMask, (*C.XEvent)(unsafe.Pointer(&evt)))

	// xdotool doesn't know if this is needed.
	// https://github.com/jordansissel/xdotool/blob/33092d8a74d60c9ad3ab39c4f05b90e047ea51d8/xdo.c#L1103-L1104
	//
	// It is. Otherwise, we'll end up with an event lingering in the X event queue,
	// and this ends up with an off-by-one error if we send multiple key events in quick succession,
	// and a spurious key event at program exit?
	C.XFlush(x.display)

	return nil
}
