package lcf

import (
	"bytes"
	"runtime"
	"time"

	"github.com/Sirupsen/logrus"
)

const (
	// Basic template just logs the level name, name field, message and fields.
	Basic = "%[levelName]s:%[name]s:%[message]s%[fields]s\n"

	// Message template just logs the message.
	Message = "%[message]s\n"

	// Detailed template logs padded columns including the running PID.
	Detailed = "%[ascTime]s %-5[process]d %-7[levelName]s %-20[name]s %[message]s%[fields]s\n"

	// DefaultTimestampFormat is the default format used if the user does not specify their own.
	DefaultTimestampFormat = "2006-01-02 15:04:05.000"
)

// CustomFormatter is the main formatter for the library.
type CustomFormatter struct {
	// Post-processed formatting template (e.g. "%s:%s:%s\n").
	Template string

	// Handler functions whose indexes match up with Template Sprintf explicit argument indexes.
	Handlers []Handler

	// Attribute names (e.g. "levelName") used in pre-processed Template.
	Attributes Attributes

	// Set to true to bypass checking for a TTY before outputting colors.
	ForceColors bool

	// Force disabling colors and bypass checking for a TTY.
	DisableColors bool

	// Timestamp format %[ascTime]s will use for display when a full timestamp is printed.
	TimestampFormat string

	// The fields are sorted by default for a consistent output. For applications
	// that log extremely frequently this may not be desired.
	DisableSorting bool

	// Different colors for different log levels.
	ColorDebug int
	ColorInfo  int
	ColorWarn  int
	ColorError int
	ColorFatal int
	ColorPanic int

	handleColors [][3]int
	startTime    time.Time
}

// Format is called by logrus and returns the formatted string.
func (f *CustomFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	// Call handlers.
	values := make([]interface{}, len(f.Handlers))
	for i, handler := range f.Handlers {
		value, err := handler(entry, f)
		if err != nil {
			return nil, err
		}
		values[i] = value
	}

	// Parse template and return.
	parsed := f.Sprintf(values...)
	return bytes.NewBufferString(parsed).Bytes(), nil
}

// NewFormatter creates a new CustomFormatter, sets the Template string, and returns its pointer.
// This function is usually called just once during a running program's lifetime.
//
// :param template: Pre-processed formatting template (e.g. "%[message]s\n").
//
// :param custom: User-defined formatters evaluated before built-in formatters. Keys are attributes to look for in the
// 	formatting string (e.g. "%[myFormatter]s") and values are formatting functions.
func NewFormatter(template string, custom CustomHandlers) *CustomFormatter {
	formatter := CustomFormatter{
		ColorDebug:      AnsiCyan,
		ColorInfo:       AnsiGreen,
		ColorWarn:       AnsiYellow,
		ColorError:      AnsiRed,
		ColorFatal:      AnsiMagenta,
		ColorPanic:      AnsiMagenta,
		TimestampFormat: DefaultTimestampFormat,
		startTime:       time.Now(),
	}

	// Parse the template string.
	formatter.ParseTemplate(template, custom)

	// Disable colors if not supported.
	if !logrus.IsTerminal() || (runtime.GOOS == "windows" && !WindowsNativeANSI()) {
		formatter.DisableColors = true
	}

	return &formatter
}
