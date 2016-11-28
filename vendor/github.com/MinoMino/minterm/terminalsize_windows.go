package minterm

import (
	"syscall"
	"unsafe"
)

type coord struct {
	x int16
	Y int16
}

type consoleScreenBufferInfo struct {
	size           coord
	cursorPosition coord
	attributes     uint16
	window         struct {
		left   int16
		top    int16
		right  int16
		bottom int16
	}
	maximumWindowSize coord
}

var getConsoleScreenBufferInfo = syscall.NewLazyDLL("kernel32.dll").NewProc("GetConsoleScreenBufferInfo")

// Returns the terminal's number of columns and rows. If something goes wrong,
// err will be non-nil, but also with reasonable fallback values of (80, 24).
// In other words, the error can often be discarded.
func TerminalSize() (columns, rows int, err error) {
	// Reasonable fallback numbers, allowing the caller to discard
	// the error without things blowing up.
	columns = 80
	rows = 24

	var csbi consoleScreenBufferInfo
	handle, err := syscall.GetStdHandle(syscall.STD_OUTPUT_HANDLE)
	if err != nil {
		return
	}

	r1, _, lastErr := getConsoleScreenBufferInfo.Call(uintptr(handle), uintptr(unsafe.Pointer(&csbi)))
	if r1 == 0 {
		if lastErr == nil {
			err = syscall.EINVAL
		}
		err = lastErr
		return
	}

	columns = int(csbi.window.right - csbi.window.left + 1)
	rows = int(csbi.window.bottom - csbi.window.top + 1)
	return
}
