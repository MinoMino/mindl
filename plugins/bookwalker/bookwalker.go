package bookwalker

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
	"bytes"
	"fmt"
	"image/jpeg"
	"image/png"
	"math/rand"
	"net/http"
	"net/url"
	"path/filepath"
	"regexp"
	"strconv"
	"time"

	"github.com/MinoMino/mindl/logger"
	"github.com/MinoMino/mindl/plugins"
	"golang.org/x/text/unicode/norm"
)

const name = "BookWalker"

var log = logger.GetLog(name)

var Plugin = BookWalker{
	options: []plugins.Option{
		&plugins.StringOption{K: "Username", Required: true},
		&plugins.StringOption{K: "Password", Required: true},
		&plugins.BoolOption{K: "Lossless", V: false,
			C: "If set to true, save as PNG. Original images are in JPEG, so you can't escape some artifacts even with this on."},
		&plugins.IntOption{K: "JPEGQuality", V: 95,
			C: "Does nothing if Lossless is on. >95 not adviced, as it increases file size a ton with little improvement."},
		&plugins.BoolOption{K: "Metadata", V: true},
	},
}

const (
	urlApi         = "https://viewer.bookwalker.jp/browserWebApi"
	urlLoginScreen = "https://member.bookwalker.jp/app/03/login"
	urlLogin       = "https://member.bookwalker.jp/app/j_spring_security_check"
	urlLogout      = "https://member.bookwalker.jp/app/03/logout"

	browserIdSuffix = "NFBR"
)

var urlBookLive, _ = url.ParseRequestURI("https://member.bookwalker.jp/")

var reBook = regexp.MustCompile(`^https?://bookwalker.jp/de(?P<cid>[a-zA-Z0-9]+?-[a-zA-Z0-9]+?-[a-zA-Z0-9]+?-[a-zA-Z0-9]+?-[a-zA-Z0-9]+?)(?:/.*)?$`)
var reReader = regexp.MustCompile(`^https?://booklive.jp/bviewer/\?cid=(?P<cid>[_0-9]+)`)
var reTokenSearch = regexp.MustCompile(`input type="hidden" name="token" value="(.+?)">`)
var reProfile = regexp.MustCompile(`^https?://member.bookwalker.jp/app/03/my/profile`)

func init() {
	// Otherwise we have deterministic generation of the browser ID.
	rand.Seed(time.Now().UnixNano())
}

type BookWalker struct {
	options []plugins.Option
	client  *http.Client
	session *BookSession
	config  *BookConfig
	content []*BookContent
}

func (bw *BookWalker) Name() string {
	return "BookWalker"
}

func (bw *BookWalker) Version() string {
	return ""
}

func (bw *BookWalker) CanHandle(url string) bool {
	return (reBook.MatchString(url) || reReader.MatchString(url))
}

func (bw *BookWalker) Options() []plugins.Option {
	return bw.options
}

func (bw *BookWalker) DownloadGenerator(url string) (dlgen func() plugins.Downloader, length int) {
	// Initialization.
	var ext string
	opts := plugins.OptionsToMap(bw.options)
	if opts["Lossless"].(bool) {
		ext = "png"
	} else {
		ext = "jpg"
	}

	// Make a client and log in.
	cid := reBook.FindStringSubmatch(url)[1]
	bw.client = plugins.NewHTTPClient(20)
	log.Info("Logging in...")
	bw.login(opts["Username"].(string), opts["Password"].(string))

	// Try to get a book session.
	var err error
	bw.session, err = bw.getBookSession(cid)
	if err != nil {
		panic(err)
	}
	dir := bw.session.Title
	dir = norm.NFKC.String(dir)

	// Get content info.
	bw.config, bw.content, err = bw.getContentInfo()
	if err != nil {
		panic(err)
	}
	length = len(bw.content)

	// Initialize descrambler.
	ds := descrambler{}

	i := 0
	// Generator.
	dlgen = func() plugins.Downloader {
		if i >= length {
			return nil
		}

		i++
		// Downloader
		return func(n int, rep plugins.Reporter) error {
			page := n + 1
			// Each file has a list of pages. I have yet to see a file with multiple
			// pages (which I call subpages), so virtually always it will have just
			// have 1 subpage.
			for _, p := range bw.content[n].FileLinkInfo.PageLinkInfoList {
				r, err := bw.getImage(page, p.Page.No)
				if err != nil {
					return err
				}
				defer r.Close()

				buf := &bytes.Buffer{}
				// Download through the reporter.
				if _, err := rep.Copy(buf, r); err != nil {
					return err
				}

				filePath := bw.content[n].FilePath + "/" + strconv.Itoa(p.Page.No)
				img, err := ds.Descramble(filePath, buf, p.Page.DummyWidth, p.Page.DummyHeight)
				var path string
				if p.Page.No > 0 {
					path = filepath.Join(dir, fmt.Sprintf("%04d-%d.%s", n+1, p.Page.No, ext))
				} else {
					path = filepath.Join(dir, fmt.Sprintf("%04d.%s", n+1, ext))
				}
				if opts["Lossless"].(bool) {
					// Save as PNG.
					w, err := rep.FileWriter(path, false)
					if err != nil {
						panic(err)
					}
					defer w.Close()

					enc := png.Encoder{}
					if err := enc.Encode(w, img); err != nil {
						return err
					}
				} else {
					// Save as JPEG.
					w, err := rep.FileWriter(path, false)
					if err != nil {
						panic(err)
					}
					defer w.Close()
					if err := jpeg.Encode(w, img, &jpeg.Options{Quality: opts["JPEGQuality"].(int)}); err != nil {
						return err
					}
				}
			}

			return nil
		}

	}
	return
}

func (bw *BookWalker) Cleanup(err error) {
	log.Info("Logging out...")
	bw.logout()
}
