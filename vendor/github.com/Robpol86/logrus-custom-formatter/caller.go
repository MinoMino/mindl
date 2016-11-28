package lcf

import (
	"runtime"
	"strings"
)

// CallerName returns the name of the calling function using the runtime package. Empty string if something fails.
//
// :param skip: Skip these many calls in the stack.
func CallerName(skip int) string {
	if pc, _, _, ok := runtime.Caller(skip); ok {
		split := strings.Split(runtime.FuncForPC(pc).Name(), ".")
		return split[len(split)-1]
	}
	return ""
}
