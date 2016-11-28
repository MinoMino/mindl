package lcf

import (
	"syscall"
)

// EnableVirtualTerminalProcessing indicates the platform supports VT100 color control character sequences.
const EnableVirtualTerminalProcessing = 0x0004

type sysCaller interface {
	getStdHandle() error
	getConsoleMode(*uint32) error
	setConsoleMode(uintptr) (uintptr, error)
}

type sysCall struct {
	nStdHandle int
	handle     syscall.Handle
}

func (s *sysCall) getStdHandle() (err error) {
	s.handle, err = syscall.GetStdHandle(s.nStdHandle)
	return
}

func (s *sysCall) getConsoleMode(mode *uint32) error {
	return syscall.GetConsoleMode(s.handle, mode)
}

func (s *sysCall) setConsoleMode(mode uintptr) (r1 uintptr, err error) {
	proc := syscall.MustLoadDLL("kernel32").MustFindProc("SetConsoleMode")
	r1, _, err = proc.Call(uintptr(s.handle), mode)
	return
}

// Does this console window have ENABLE_VIRTUAL_TERMINAL_PROCESSING enabled? Optionally try to enable if not.
func windowsNativeANSI(stderr bool, setMode bool, sc sysCaller) (enabled bool, err error) {
	if sc == nil {
		if stderr {
			sc = &sysCall{nStdHandle: syscall.STD_ERROR_HANDLE}
		} else {
			sc = &sysCall{nStdHandle: syscall.STD_OUTPUT_HANDLE}
		}
	}

	// Get win32 handle.
	if err = sc.getStdHandle(); err != nil {
		return
	}

	// Get console mode.
	var dwMode uint32
	if err = sc.getConsoleMode(&dwMode); err != nil {
		return
	}
	enabled = dwMode&EnableVirtualTerminalProcessing != 0
	if enabled || !setMode {
		return
	}

	// Try to enable the feature.
	dwMode |= EnableVirtualTerminalProcessing
	if r1, err := sc.setConsoleMode(uintptr(dwMode)); r1 == 0 {
		return false, err
	}
	return true, nil
}
