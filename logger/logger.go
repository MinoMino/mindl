package logger

import (
	"fmt"
	"os"

	log "github.com/MinoMino/logrus"
	lcf "github.com/MinoMino/logrus-custom-formatter"
)

// A cute little helper struct that forces the writer to
// get the value of os.Stdout every time it writes.
// Setting this as the output for the logger makes sure that
// when we change os.Stdout with minterm.LineReserver, it'll
// properly output through the replaced os.Stdout instead of
// the actual one it saved during package initialization.
type stdoutReferer struct {
	stdout **os.File
}

func (std *stdoutReferer) Write(p []byte) (int, error) {
	w := *std.stdout
	return w.Write(p)
}

type Fields map[string]interface{}

func init() {
	NameHandler := func(e *log.Entry, f *lcf.CustomFormatter) (interface{}, error) {
		if n, ok := e.Data["name"]; ok {
			return fmt.Sprintf("[%s] ", n), nil
		}

		return "", nil
	}

	std := &stdoutReferer{&os.Stdout}
	log.SetOutput(std)
	templ := "(%[ascTime]s %[shortLevelName]s) %[name]s%-45[message]s%[fields]s\n"
	formatter := lcf.NewFormatter(templ, lcf.CustomHandlers{"name": NameHandler})
	formatter.TimestampFormat = "15:04:05"
	log.SetFormatter(formatter)
}

func Verbose(enable bool) {
	if enable {
		log.SetLevel(log.DebugLevel)
	}
}

func GetLog(name string) log.FieldLogger {
	if name == "" {
		return log.StandardLogger()
	}

	return log.WithField("name", name)
}
