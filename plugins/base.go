package plugins

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
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	log "github.com/MinoMino/logrus"
)

/*
   ==================================================
                          MISC
    Helpers and stuff that can be useful for plugins.
   ==================================================
*/

const (
	FirefoxUserAgent = "Mozilla/5.0 (compatible; MSIE 9.0; Windows NT 6.1; Trident/5.0)"
	IE11UserAgent    = "Mozilla/5.0 (Windows NT 6.1; WOW64; Trident/7.0; rv:11.0) like Gecko"
	ChromeUserAgent  = "Mozilla/5.0 (Windows NT 6.3; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/54.0.2840.99 Safari/537.36"
	SafariUserAgent  = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_9_3) AppleWebKit/537.75.14 (KHTML, like Gecko) Version/7.0.3 Safari/7046A194A"
)

// Implements the error interface.
type ErrHTTPStatusCode struct {
	StatusCode int
}

func (e *ErrHTTPStatusCode) String() string {
	return "The HTTP request did not respond with status code 200."
}

// Panic with an ErrHTTPStatusCode if the status code isn't 200.
func PanicForStatus(resp *http.Response, msg string) {
	if resp.StatusCode != http.StatusOK {
		if msg != "" {
			msg = " | " + msg
		}
		log.Errorf("Status code: %s%s", resp.Status, msg)
		panic(&ErrHTTPStatusCode{resp.StatusCode})
	}
}

// Convert a slice of Option into a map[string]interface{}.
func OptionsToMap(opts []Option) map[string]interface{} {
	res := make(map[string]interface{})
	for _, opt := range opts {
		res[opt.Key()] = opt.Value()
	}

	return res
}

// Create an HTTP client with a proper timeout timer.
func NewHTTPClient(timeout int) *http.Client {
	jar, _ := cookiejar.New(nil)
	return &http.Client{
		Timeout: time.Second * 20,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			last := via[len(via)-1]
			log.WithField("url", last.URL.String()).Debug("Following HTTP redirect...")
			req.Header = last.Header
			if req.URL.Host != last.URL.Host {
				delete(req.Header, "Authorization")
			}

			return nil
		},
		Jar: jar,
	}
}

// Create a new GET request with a Firefox user agent.
func NewGetRequest(url string) *http.Request {
	return NewGetRequestUA(url, FirefoxUserAgent)
}

// Create a new GET request with a custom user agent.
func NewGetRequestUA(url, userAgent string) *http.Request {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Error("Error while creating a new GET request.")
		panic(err)
	}
	req.Header.Set("User-Agent", userAgent)

	return req
}

// Create a new POST request with a Firefox user agent using form data.
func NewPostFormRequest(url string, data url.Values) *http.Request {
	return NewPostFormRequestUA(url, FirefoxUserAgent, data)
}

// Create a new POST request with a Firefox user agent using form data.
func NewPostFormRequestUA(url, userAgent string, data url.Values) *http.Request {
	req, err := http.NewRequest("POST", url, strings.NewReader(data.Encode()))
	if err != nil {
		log.Error("Error while creating a new POST request.")
		panic(err)
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	return req
}

/*
   ==================================================
                        REPORTER
   ==================================================
*/

// An interface passed to Downloader that allows it to get a destination
// to write downloads to and to get temporary files. Using this interface
// instead of opening files inside the plugin allows the download manager
// to control the flow of downloads (e.g. pausing, aborting), track download
// speeds, stay aware of downloaded files for further processing, and so on.
//
// All paths used to save files with must relative and be in a directory.
// If the user wants to have the files zipped, all the top-level directories
// will be zipped.
type Reporter interface {
	// Should be used when you have something you need to download, but not
	// actually something you want the manager to save and count as part of
	// the resulting files.
	//
	// This doesn't directly benefit the downloader, but it allows the manager
	// to keep track of download speeds, and more importantly if something were
	// to go wrong in another downloader, it can detect that and stop the downloader
	// from keeping the application from exiting.
	Copy(dst io.Writer, src io.Reader) (written int64, err error)
	// Saves the data/file into a file as a successful download.
	// The path must be relative, as the downloader will take care of where to save files.
	// The report bool determines whether or not it should report download speeds.
	// In other words, whether or not src is getting its data straight from the network.
	SaveData(dst string, src io.Reader, report bool) (written int64, err error)
	// Saves the file as a successful download. The destination path must be relative,
	// as the downloader will take care of where to save files. The file is renamed (moved)
	// to the final destination rather than copied. This only works if the file resides on
	// the same disk drive as the the destination, so use SaveFile() if unsure.
	// Should be safe to use with TempFile() files. src must be closed before using this.
	SaveFile(dst, src string) (size int64, err error)
	// For when you need a temporary file. Should ensure the file resides on the same
	// disk drive as the download directory, allowing for use with SaveFile().
	TempFile() (*os.File, error)
	// Returns a writer to the destination file. The caller must close it.
	// Download completion is reported on close.
	FileWriter(dst string, report bool) (io.WriteCloser, error)
}

/*
   ==================================================
                         OPTION
     The section where you wish you had templates.
   ==================================================
*/

// An option the plugin provides that the user can set.
// The input is through a string provided by the user.
type Option interface {
	Key() string
	Value() interface{}
	// Set the value using user input.
	Set(string) error
	IsRequired() bool
	IsHidden() bool
	Comment() string
}

// A basic Option implementation that keeps all user
// input as-is instead of trying to convert stuff.
type StringOption struct {
	K, V             string
	Required, Hidden bool
	C                string
}

func (opt *StringOption) Key() string {
	return opt.K
}

func (opt *StringOption) Value() interface{} {
	return opt.V
}

func (opt *StringOption) Set(v string) error {
	opt.V = v
	return nil
}

func (opt *StringOption) IsRequired() bool {
	return opt.Required
}

func (opt *StringOption) IsHidden() bool {
	return opt.Hidden
}

func (opt *StringOption) Comment() string {
	return opt.C
}

// An implementation of Option that tries to convert
// the user input into an integer.
type IntOption struct {
	K                string
	V                int
	Required, Hidden bool
	C                string
}

func (opt *IntOption) Key() string {
	return opt.K
}

func (opt *IntOption) Value() interface{} {
	return opt.V
}

func (opt *IntOption) Set(v string) (err error) {
	opt.V, err = strconv.Atoi(v)
	return err
}

func (opt *IntOption) IsRequired() bool {
	return opt.Required
}

func (opt *IntOption) IsHidden() bool {
	return opt.Hidden
}

func (opt *IntOption) Comment() string {
	return opt.C
}

// An implementation of Option that tries to convert
// the user input into a float64.
type FloatOption struct {
	K                string
	V                float64
	Required, Hidden bool
	C                string
}

func (opt *FloatOption) Key() string {
	return opt.K
}

func (opt *FloatOption) Value() interface{} {
	return opt.V
}

func (opt *FloatOption) Set(v string) (err error) {
	opt.V, err = strconv.ParseFloat(v, 64)
	return err
}

func (opt *FloatOption) IsRequired() bool {
	return opt.Required
}

func (opt *FloatOption) IsHidden() bool {
	return opt.Hidden
}

func (opt *FloatOption) Comment() string {
	return opt.C
}

// An implementation of Option that tries to convert
// the user input into a bool. Using strconv.ParseBool,
// it accepts 1, t, T, TRUE, true, True, 0, f, F, FALSE,
// false, False.
type BoolOption struct {
	K                string
	V                bool
	Required, Hidden bool
	C                string
}

func (opt *BoolOption) Key() string {
	return opt.K
}

func (opt *BoolOption) Value() interface{} {
	return opt.V
}

func (opt *BoolOption) Set(v string) (err error) {
	opt.V, err = strconv.ParseBool(v)
	return err
}

func (opt *BoolOption) IsRequired() bool {
	return opt.Required
}

func (opt *BoolOption) IsHidden() bool {
	return opt.Hidden
}

func (opt *BoolOption) Comment() string {
	return opt.C
}

// An option to force the download manager to either zip or not zip the directories
// after the download finishes.
type ForceZipOption struct {
	BoolOption
}

func NewForceZipOption(force bool) *ForceZipOption {
	return &ForceZipOption{
		BoolOption{
			V: force,
		},
	}
}

func (opt *ForceZipOption) Key() string {
	return "!Zip"
}

func (opt *ForceZipOption) IsRequired() bool {
	return false
}

func (opt *ForceZipOption) IsHidden() bool {
	return true
}

func (opt *ForceZipOption) Comment() string {
	return "Force the download manager to zip the directories after the download finishes."
}

// An option to force the number of workers used by the download manager.
type MaxWorkersOption struct {
	IntOption
}

func NewForceMaxWorkersOption(workers int) *MaxWorkersOption {
	return &MaxWorkersOption{
		IntOption{
			V: workers,
		},
	}
}

func (opt *MaxWorkersOption) Key() string {
	return "!Workers"
}

func (opt *MaxWorkersOption) IsRequired() bool {
	return false
}

func (opt *MaxWorkersOption) IsHidden() bool {
	return true
}

func (opt *MaxWorkersOption) Comment() string {
	return "Force the maximum number of workers to a certain number."
}

/*
   ==================================================
                         PLUGIN
   ==================================================
*/

const UnknownTotal = 0

type Downloader func(int, Reporter) error

// The interface all plugins must implement.
//
// A single object is used for multiple URLs that concern the plugin, so the
// implementation must deal with a call to DownloadGenerator() as a new
// download and reset any variables and whatnot that could affect the result.
type Plugin interface {
	// The name of the plugin.
	Name() string
	// The version of the plugin. Do not prefix it with "v" or anything like that.
	// Can return an empty string.
	Version() string
	// Should return whether or not it can deal with a URL.
	// There is no guarantee that DownloadGenerator() will be called later.
	// There is also no guarantee that the last call to this is the URL that
	// will be passed to DownloadGenerator(). In other words, don't store stuff
	// here for later use.
	CanHandle(url string) bool
	// Returns a slice of all the options. If the user wants to, they can
	// be set. Of course, if an option needs a value and the user does not
	// set it, either make sure you have a default value or that IsRequired()
	// returns true. The latter will not allow the plugin to run without setting it.
	Options() []Option
	// A generator function that returns a generator that generates downloaders.
	// Sounds overly convoluted, but when combined with closures, it's pretty damn neat.
	// The outer function could theoretically be dropped and just have it directly generate
	// Downloader functions, but if you have to initialize stuff first, you'd have to check
	// whether or not you've initialized at the start of the function for every call.
	// With this signature, you can just initialize and return the generator as a closure to
	// keep all the local variables and not have to store them anywhere first. This also
	// makes it ideal for having one single object per plugin without ever having to remake it.
	//
	// The "dls" is the number of downloaders that are going to be returned. Use UnknownTotal
	// if it's unknown. You can go over or under the total without it breaking anything important.
	// It's simply used for displaying progress through the interface.
	//
	// See the Dummy plugin for an example implementation.
	DownloadGenerator(url string) (dlgen func() Downloader, dls int)
	// A method called by the download manager at the end. If an error caused the manager
	// to abort, it is passed. Otherwise nil is passed.
	Cleanup(error)
}
