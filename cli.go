package main

// mindl - A downloader for various sites and services.
// Copyright (C) 2016  Mino <mino@minomino.org>
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

import (
	"errors"
	//"flag"
	"fmt"
	"os"
	"strings"
	"time"

	flag "github.com/spf13/pflag"

	. "github.com/MinoMino/mindl/plugins"
	"github.com/MinoMino/minterm"
	lcf "github.com/Robpol86/logrus-custom-formatter"
	log "github.com/Sirupsen/logrus"
)

// Set by make on compilation.
var version = "UNSET"

// Errors.
var (
	ErrInvalidOptionFormat = errors.New("Invalid option format. Should be key=value.")
)

// Flag for options passed through the CLI that satisfies
// the flag.Getter interface.
type OptionsFlag map[string]string

func (opt *OptionsFlag) Get() interface{} {
	return *opt
}

func (opt *OptionsFlag) String() string {
	res := make([]string, 0, 5)
	for k, v := range map[string]string(*opt) {
		res = append(res, fmt.Sprintf("%q: %q", k, v))
	}

	content := strings.Join(res, ", ")
	if content != "" {
		return fmt.Sprintf("{%s}", content)
	}

	return ""
}

func (opt *OptionsFlag) Set(v string) error {
	split := strings.SplitN(v, "=", 2)
	if len(split) < 2 {
		return ErrInvalidOptionFormat
	}

	if *opt == nil {
		*opt = OptionsFlag(make(map[string]string))
	}
	(*opt)[split[0]] = split[1]
	return nil
}

func (opt *OptionsFlag) Type() string {
	return "key=value"
}

var (
	options                     OptionsFlag
	workers                     int
	verbose, defaults, noprompt bool
	directory                   string
	urls                        []string
)

func init() {
	flag.VarP(&options, "option", "o",
		"Options in a key=value format passed to plugins.")
	flag.IntVarP(&workers, "workers", "w", 10,
		"The number of workers to use.")
	flag.BoolVarP(&verbose, "verbose", "v", false,
		"Set to display debug messages.")
	flag.BoolVarP(&defaults, "defaults", "d", false,
		"Set to use default values for options whenever possible. No effect if --no-prompt is on.")
	flag.BoolVarP(&noprompt, "no-prompt", "n", false,
		"Set to turn off prompts for options and instead throw an error if a required option is left unset.")
	flag.StringVarP(&directory, "directory", "D", "downloads/",
		"The directory in which to save the downloaded files.")
}

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

func init() {
	std := &stdoutReferer{&os.Stdout}
	log.SetOutput(std)
	template := "%[shortLevelName]s[%04[relativeCreated]d] %-45[message]s%[fields]s\n"
	log.SetFormatter(lcf.NewFormatter(template, nil))
}

func main() {
	flag.Parse()
	urls = flag.Args()
	if verbose {
		log.SetLevel(log.DebugLevel)
	}

	if flag.NArg() == 0 {
		flag.Usage()
		os.Exit(0)
	}

	pm := PluginManager(Plugins[:])
	handlers := pm.FindHandlers(urls)
	for i, h := range handlers {
		// Ensure we have at least one handler for each URL.
		if len(h) == 0 {
			log.Errorf("Found no handler for: %s", urls[i])
		}
		// Set options for the plugin.
		if err := pm.SetOptions(h, map[string]string(options), defaults, noprompt); err != nil {
			log.Fatal(err)
		}
	}

	// Start downloading.
	for i, h := range handlers {
		// Make the user pick a handler if multiple plugins
		// can handle a URL.
		// TODO: Make it possible to run mindl without user input.
		if p, err := pm.SelectPlugin(h); err != nil {
			log.Fatal(err)
		} else {
			// If we're dealing with multiple URLs, print which one we're processing.
			if len(urls) > 1 {
				log.Infof("Processing URL: %s", urls[i])
			}
			log.Infof("Starting download using \"%s\"...", pluginName(p))
			startDownloading(urls[i], p)
		}
	}
}

func startDownloading(url string, plugin Plugin) {
	dm := NewDownloadManager(plugin)
	lr, _ := minterm.NewLineReserver()
	defer func() {
		if r := recover(); r != nil {
			log.Fatalf("Panicked: %v", r)
		}
	}()
	defer lr.Release()

	// Get a new progress string and refresh the reserved line
	// in regular intervals.
	ticker := time.NewTicker(time.Millisecond * 500)
	done := make(chan struct{})
	defer func() {
		ticker.Stop()
		done <- struct{}{}
	}()
	go func() {
		for {
			select {
			case <-ticker.C:
				lr.Set(dm.ProgressString())
				lr.Refresh()
			case <-done:
				return
			}
		}
	}()

	dls, err := dm.Download(url, workers)
	if err != nil {
		log.Error(err)
		return
	}
	log.Infof("Done! Got a total of %d downloads.", len(dls))
}
