package lcf

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
)

var _reBracketed = regexp.MustCompile(`%([\d.-]*)\[(\w+)](\w)`)

// Handler is the function signature of formatting attributes such as "levelName" and "message".
type Handler func(*logrus.Entry, *CustomFormatter) (interface{}, error)

// CustomHandlers is a mapping of Handler-type functions to attributes as key names (e.g. "levelName").
//
// With this type many custom handler functions can be defined and fed to NewFormatter(). CustomHandlers are parsed
// first so you can override built-in handlers such as the one for %[ascTime]s with your own. Since they are exported
// you can call built-in handlers in your own custom handler. The returned interface{} value is passed to fmt.Sprintf().
//
// In addition to overriding handlers you can create new attributes (such as %[myAttr]s) and map it to your Handler
// function.
type CustomHandlers map[string]Handler

// Attributes is a map used like a "set" to keep track of which formatting attributes are used.
type Attributes map[string]bool

// Contains returns true if attr is present.
func (a Attributes) Contains(attr string) bool {
	_, ok := a[attr]
	return ok
}

// HandlerAscTime returns the formatted timestamp of the entry.
func HandlerAscTime(entry *logrus.Entry, formatter *CustomFormatter) (interface{}, error) {
	return entry.Time.Format(formatter.TimestampFormat), nil
}

// HandlerFields returns the entry's fields (excluding name field if %[name]s is used) colorized according to log level.
// Fields' formatting: key=value key2=value2
func HandlerFields(entry *logrus.Entry, formatter *CustomFormatter) (interface{}, error) {
	var fields string

	// Without sorting no need to get keys from map into a string array.
	if formatter.DisableSorting {
		for key, value := range entry.Data {
			if key == "name" && formatter.Attributes.Contains("name") {
				continue
			}
			fields = fmt.Sprintf("%s %s=%v", fields, Color(entry, formatter, key), value)
		}
		return fields, nil
	}

	// Put keys in a string array and sort it.
	keys := make([]string, len(entry.Data))
	i := 0
	for k := range entry.Data {
		keys[i] = k
		i++
	}
	sort.Strings(keys)

	// Do the rest.
	for _, key := range keys {
		if key == "name" && formatter.Attributes.Contains("name") {
			continue
		}
		fields = fmt.Sprintf("%s %s=%v", fields, Color(entry, formatter, key), entry.Data[key])
	}
	return fields, nil
}

// HandlerLevelName returns the entry's long level name (e.g. "WARNING").
func HandlerLevelName(entry *logrus.Entry, formatter *CustomFormatter) (interface{}, error) {
	return Color(entry, formatter, strings.ToUpper(entry.Level.String())), nil
}

// HandlerName returns the name field value set by the user in entry.Data.
func HandlerName(entry *logrus.Entry, _ *CustomFormatter) (interface{}, error) {
	if value, ok := entry.Data["name"]; ok {
		return value.(string), nil
	}
	return "", nil
}

// HandlerMessage returns the unformatted log message in the entry.
func HandlerMessage(entry *logrus.Entry, _ *CustomFormatter) (interface{}, error) {
	return entry.Message, nil
}

// HandlerProcess returns the current process' PID.
func HandlerProcess(_ *logrus.Entry, _ *CustomFormatter) (interface{}, error) {
	return os.Getpid(), nil
}

// HandlerRelativeCreated returns the number of seconds since program start time.
func HandlerRelativeCreated(_ *logrus.Entry, formatter *CustomFormatter) (interface{}, error) {
	return int(time.Since(formatter.startTime) / time.Second), nil
}

// HandlerShortLevelName returns the first 4 letters of the entry's level name (e.g. "WARN").
func HandlerShortLevelName(entry *logrus.Entry, formatter *CustomFormatter) (interface{}, error) {
	return Color(entry, formatter, strings.ToUpper(entry.Level.String()[:4])), nil
}

// ParseTemplate parses the template string and prepares it for fmt.Sprintf() and keeps track of which handlers to use.
//
// :param template: Pre-processed formatting template (e.g. "%[message]s\n").
//
// :param custom: User-defined formatters evaluated before built-in formatters. Keys are attributes to look for in the
func (f *CustomFormatter) ParseTemplate(template string, custom CustomHandlers) {
	f.Attributes = make(Attributes)
	segments := []string{}
	segmentsPos := 0

	for _, idxs := range _reBracketed.FindAllStringSubmatchIndex(template, -1) {
		// Find attribute names to replace and with what handler function to map them to.
		attribute := template[idxs[4]:idxs[5]]
		if fn, ok := custom[attribute]; ok {
			f.Handlers = append(f.Handlers, fn)
		} else {
			switch attribute {
			case "ascTime":
				f.Handlers = append(f.Handlers, HandlerAscTime)
			case "fields":
				f.Handlers = append(f.Handlers, HandlerFields)
			case "levelName":
				f.Handlers = append(f.Handlers, HandlerLevelName)
			case "name":
				f.Handlers = append(f.Handlers, HandlerName)
			case "message":
				f.Handlers = append(f.Handlers, HandlerMessage)
			case "process":
				f.Handlers = append(f.Handlers, HandlerProcess)
			case "relativeCreated":
				f.Handlers = append(f.Handlers, HandlerRelativeCreated)
			case "shortLevelName":
				f.Handlers = append(f.Handlers, HandlerShortLevelName)
			default:
				continue
			}
		}
		f.Attributes[attribute] = true

		// Add segments of the template that do not match regexp (between attributes).
		if segmentsPos < idxs[0] {
			segments = append(segments, template[segmentsPos:idxs[0]])
		}

		// Keep track of padded (y-x > 0) string (== 's') attributes for ANSI color handling.
		if template[idxs[6]:idxs[7]] == "s" && idxs[3]-idxs[2] > 0 {
			start := 0
			for _, s := range segments {
				start += len(s)
			}
			end := start + idxs[3] - idxs[0] + idxs[7] - idxs[6]
			f.handleColors = append(f.handleColors, [...]int{len(f.Handlers) - 1, start, end})
		}

		// Update segments.
		segments = append(segments, template[idxs[0]:idxs[3]]+template[idxs[6]:idxs[7]])
		segmentsPos = idxs[1]
	}

	// Add trailing segments of the template that did not match the regexp (newline).
	if segmentsPos < len(template) {
		segments = append(segments, template[segmentsPos:])
	}

	// Join segments.
	f.Template = strings.Join(segments, "")
}
