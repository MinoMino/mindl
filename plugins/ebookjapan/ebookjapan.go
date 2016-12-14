package ebookjapan

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
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/MinoMino/mindl/logger"
	"github.com/MinoMino/mindl/plugins"
	"github.com/sclevine/agouti"
	"golang.org/x/text/unicode/norm"
)

const name = "eBookJapan"

var log = logger.GetLog(name)

const (
	// How many seconds to wait for the page to load for.
	loadTimeout = 20.0
	// How many seconds to wait for the page data to be returned.
	dataTimeout = 10.0
	// How many milliseconds to wait before polling again.
	loadPolling = 250
	dataPolling = 500
	// How many pages we rip before we reopen the reader.
	reopenCount = 50
)

var (
	ErrEBJPhantomJSNotFound = errors.New("Could not find the PhantomJS executable.")
	ErrEBJNoLoad            = errors.New("The reader did not load nor raise any errors.")
	ErrEBJNoData            = errors.New("Page data did not return before the time limit.")
)

var Plugin = EBookJapan{
	[]plugins.Option{
		&plugins.BoolOption{K: "Lossless", V: false, Required: false,
			C: "If set to true, save as PNG. Original images are in JPEG, so you can't escape some artifacts even with this on."},
		&plugins.IntOption{K: "JPEGQuality", V: 95,
			C: "Does nothing if Lossless is on. >95 not adviced, as it increases file size a ton for little improvement."},
		&plugins.IntOption{K: "PrefetchCount", V: 5,
			C: "How many pages should be prefetched. The higher, the faster downloads, but also more RAM and CPU usage."},
	},
}

var ebjUrlRegex = regexp.MustCompile(`^https?://br.ebookjapan.jp/br/reader/viewer/view.html\?.+$`)

type EBookJapan struct {
	options []plugins.Option
}

func (ebj *EBookJapan) Name() string {
	return "EBookJapan"
}

func (ebj *EBookJapan) Version() string {
	return ""
}

func (ebj *EBookJapan) CanHandle(url string) bool {
	return ebjUrlRegex.MatchString(url)
}

func (ebj *EBookJapan) Options() []plugins.Option {
	return ebj.options
}

func (ebj *EBookJapan) DownloadGenerator(url string) (dlgen func() plugins.Downloader, length int) {
	// Initialization.
	var ext string
	opts := plugins.OptionsToMap(ebj.options)
	if opts["Lossless"].(bool) {
		ext = "png"
	} else {
		ext = "jpg"
	}
	driver := agouti.PhantomJS()
	log.Info("Starting PhantomJS...")
	if err := driver.Start(); err != nil {
		panic("Failed to start PhantomJS: " + err.Error())
	}

	// Make a page, load the reader, then run the ripper script.
	var page *agouti.Page
	page, length = getReaderPage(driver, url, true)

	// Remove the canvases on the reader to reduce memory footprint.
	if err := page.RunScript(reduceMemoryScript, nil, nil); err != nil {
		panic(err)
	}

	// An slice of bools indicating whether or not a page is being prefetched.
	prefetched := make([]bool, length)
	prefetchCount := opts["PrefetchCount"].(int)

	// Metadata fetching.
	metadata := make(map[string]interface{})
	if err := page.RunScript(`return BR_page.jsonData.bif;`, nil, &metadata); err != nil {
		panic(err)
	}

	dir, err := page.Title()
	dir = norm.NFKC.String(dir)
	if err != nil {
		panic("Failed to get the page title: " + err.Error())
	}

	once := false
	// Generator.
	dlgen = func() plugins.Downloader {
		// Only one instance of PhantomJS and we can't do stuff concurrently
		// from the Go side of things, so only one Downloader is ever returned.
		if once {
			return nil
		}

		once = true
		return func(n int, rep plugins.Reporter) error {
			// Make sure we stop the driver before we exit.
			defer driver.Stop()

			var reopened bool
			for i := 0; i < length; i++ {
				// PhantomJS sucks and forces us to reopen the page every now and then
				// or else it'll like 1.5 GB memory and eventually crash.
				if i != 0 && i%reopenCount == 0 {
					log.Info("Closing and reopening reader...")
					// PhantomJS is shit and doesn't GC unless you close the page,
					// so to reduce memory usage and prevent it from crashing we
					// close the page and reopen it, run scripts again, etc. etc.
					if err := page.Destroy(); err != nil {
						log.Error("Failed to destroy the page.")
						panic(err)
					}

					page, _ = getReaderPage(driver, url, false)
					reopened = true
				}

				// Prefetch pages before we start polling.
				for j := 0; j < prefetchCount && j+i < length; j++ {
					// Skip if already prefetched and make sure we don't prefetch if we're
					// reopening the reader soon.
					if prefetched[i+j] {
						continue
					} else if !reopened && i != 0 && (i+j)%reopenCount == 0 {
						break
					} else if reopened {
						reopened = false
					}
					log.Debugf("Prefetching page %d...", j+i+1)
					// Asynchronously get pages.
					if err := page.RunScript(fmt.Sprintf(futureScript, j+i+1), nil, nil); err != nil {
						panic(err)
					}
					prefetched[i+j] = true
				}

				// Start polling for the data.
				var data string
				now := time.Now()
				for time.Since(now).Seconds() < dataTimeout {
					if err := page.RunScript(fmt.Sprintf(fetchDataScript, i+1), nil, &data); err != nil {
						panic(err)
					} else if data != "" {
						// We got something. Clean up and break.
						if err := page.RunScript(fmt.Sprintf(cleanupScript, i+1), nil, nil); err != nil {
							panic(err)
						}
						break
					}

					// Regulate polling speed.
					time.Sleep(time.Millisecond * dataPolling)
				}

				// Check if we got data, or for whatever reason got malformed data.
				if data == "" || len(data) < 22 {
					return ErrEBJNoData
				}

				// We have the page in base64, so all we need to do is decode it.
				dataReader := strings.NewReader(data[strings.Index(data, ",")+1:])
				dec := base64.NewDecoder(base64.StdEncoding, dataReader)
				path := filepath.Join(dir, fmt.Sprintf("%04d.%s", i+1, ext))
				// Further decode the decoded data as an image.
				img, _, err := image.Decode(dec)
				if err != nil {
					return err
				}
				// Prepare to write a file.
				w, err := rep.FileWriter(path, false)
				if err != nil {
					panic(err)
				}

				if opts["Lossless"].(bool) {
					// The data we got from the canvas is already a PNG file, but it doesn't
					// use compression at all from the looks of it. Re-encoding it massively
					// reduces file size, so it's worth the trouble.
					enc := png.Encoder{}
					if enc.Encode(w, img); err != nil {
						panic(err)
					}
				} else {
					// Save as JPEG. We could theoretically just get the file as a
					// JPEG from the canvas, but I trust this encoder more in every
					// aspect. Could still be worth to compare speeds, though.
					w, err := rep.FileWriter(path, false)
					if err != nil {
						panic(err)
					}
					if jpeg.Encode(w, img, &jpeg.Options{Quality: opts["JPEGQuality"].(int)}); err != nil {
						panic(err)
					}
				}
				w.Close()
			}

			return nil
		}

	}
	return
}

func (ebj *EBookJapan) Cleanup(err error) {

}

func waitForLoad(page *agouti.Page) error {
	now := time.Now()
	for time.Since(now).Seconds() < loadTimeout {
		if _, err := page.FindByID("canvas-0").Elements(); err != nil {
			if msg := getAlert(page); msg != "" {
				return fmt.Errorf("Found alert: %s", msg)
			}
			// No errors by the reader, so keep trying.
		} else {
			// Canvas was found, so we're good to go.
			return nil
		}

		// Regulate polling speed.
		time.Sleep(time.Millisecond * loadPolling)
	}

	return ErrEBJNoLoad
}

func getReaderPage(driver *agouti.WebDriver, url string, doLog bool) (*agouti.Page, int) {
	page, err := driver.NewPage(agouti.Browser("firefox"))
	if err != nil {
		panic("Failed to open page: " + err.Error())
	}

	if doLog {
		log.Info("Opening the reader...")
	}
	if err := page.Navigate(url); err != nil {
		panic("Failed to navigate: " + err.Error())
	}
	hookAlert(page)

	if doLog {
		log.Info("Waiting for reader to load...")
	}
	if err := waitForLoad(page); err != nil {
		panic(err)
	}

	// Main script runs here.
	var length int
	if err := page.RunScript(ripperScript, nil, &length); err != nil {
		panic(err)
	}

	return page, length
}

func hookAlert(page *agouti.Page) {
	page.RunScript(`window.alert = function(m) { _myalert = m; }`, nil, nil)
}

func getAlert(page *agouti.Page) string {
	var out string
	page.RunScript(`return _myalert`, nil, &out)

	return out
}
