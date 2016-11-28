package dummy

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

// Plugin that produces random data, but with delayed reading.
// This makes it act similar to a real download and is useful
// for trying out stuff and whatnot.

import (
	"bytes"
	"fmt"
	"io"
	"math/rand"
	"path/filepath"
	"regexp"
	"strconv"
	"time"

	"github.com/MinoMino/mindl/plugins"
)

// Using random intervals, sleeps here and there between reads.
type DelayedReader struct {
	io.Reader
	min, max int
}

func (d *DelayedReader) Read(p []byte) (int, error) {
	time.Sleep(time.Millisecond * time.Duration(rand.Intn(d.max-d.min)+d.min))
	return d.Reader.Read(p)
}

var Plugin = Dummy{
	[]plugins.Option{
		&plugins.StringOption{K: "Hello", V: "World", Required: false},
		&plugins.StringOption{K: "I Like", V: "Potatoes", Required: false},
	},
}

var dummyUrlRegex = regexp.MustCompile(`^dummy://(?P<length>\d+)$`)

type Dummy struct {
	options []plugins.Option
}

func (d *Dummy) Name() string {
	return "Dummy"
}

func (d *Dummy) Version() string {
	return ""
}

func (d *Dummy) CanHandle(url string) bool {
	return dummyUrlRegex.MatchString(url)
}

func (d *Dummy) Options() []plugins.Option {
	return d.options
}

func (d *Dummy) DownloadGenerator(url string) (dlgen func() plugins.Downloader, length int) {
	// Initialization.
	re := dummyUrlRegex.FindStringSubmatch(url)
	length, _ = strconv.Atoi(re[1])
	rand.Seed(int64(length))
	dir := fmt.Sprintf("dummy-%d", time.Now().Unix())
	i := 0

	// Generator.
	dlgen = func() plugins.Downloader {
		if i >= length {
			return nil
		}
		i++

		// Downloader. These are ran by the framework as goroutines.
		//
		// Note that all variables in these closures aren't evaluated when returned,
		// but when it's ran. This means that you can't use the "i" variable here
		// inside the downloader as a counter, as it's going to change before its
		// evaluation. However, since you are passed a counter, you can use that
		// to get data for that specific goroutine.
		return func(n int, rep plugins.Reporter) error {
			size := rand.Intn(1e6) + 1e5
			buf := make([]byte, size)
			rand.Read(buf)
			r := bytes.NewBuffer(buf)

			_, err := rep.SaveData(
				filepath.Join(dir, fmt.Sprintf("dummy-%d.bin", n)),
				&DelayedReader{r, 200, 1000},
				true)
			return err
		}

	}

	return
}
