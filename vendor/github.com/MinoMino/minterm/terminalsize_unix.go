// +build darwin dragonfly freebsd linux netbsd openbsd

package minterm

import (
	"syscall"
	"unsafe"
)

// Struct according to sys/ioctl.h.
type winsize struct {
	row    uint16
	col    uint16
	xpixel uint16
	ypixel uint16
}

// Returns the terminal's number of columns and rows. If something goes wrong,
// err will be non-nil, but also with reasonable fallback values of (80, 24).
// In other words, the error can often be discarded.
func TerminalSize() (columns, rows int, err error) {
	// Reasonable fallback numbers, allowing the caller to discard
	// the error without things blowing up.
	columns = 80
	rows = 24

	winsz := &winsize{}
	res, _, e := syscall.Syscall(syscall.SYS_IOCTL, uintptr(syscall.Stdin),
		uintptr(syscall.TIOCGWINSZ), uintptr(unsafe.Pointer(winsz)))
	if int(res) == -1 {
		err = e
		return
	}
	columns = int(winsz.col)
	rows = int(winsz.row)

	return
}
