/*
A helper package to make calls to the API served by BinB Reader.
*/
package binb

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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	log "github.com/MinoMino/logrus"
)

const userAgent = "Mozilla/5.0 (compatible; MSIE 9.0; Windows NT 6.1; Trident/5.0)"

var reTtxImagePath = regexp.MustCompile(`t-img src="(.+?)"`)
var reDataUri = regexp.MustCompile(`^(?:data:)?(?P<mime>[\w/\-\.]+);(?P<encoding>\w+),(?P<data>.*)$`)

// For k generation. Doesn't really need to be implemented like the JS, but we don't
// want to stand out, so we implement it like the JS code does.
const apiAlphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

// API methods for easy formatting of the URL.
var bibApi = map[string]string{
	"get_content_info":     "%s/bibGetCntntInfo.php?%s",
	"get_bibliography":     "%s/bibGetBibliography.php?%s",
	"get_content_settings": "%s/bibGetCntSetting.php?%s",
	"set_content_settings": "%s/bibUdtCntSetting.php?%s",
	"get_memo":             "%s/bibGetMemo.php?%s",
	"set_memo":             "%s/bibRegMemo.php?%s",
}
var sbcApi = map[string]string{
	"check_login":          "%s/sbcChkLogin.php?%s",
	"check_p":              "%s/sbcPCheck.php?%s",
	"content_check":        "%s/sbcContentCheck.php?%s",
	"get_content":          "%s/sbcGetCntnt.php?%s",
	"get_image":            "%s/sbcGetImg.php?%s",
	"get_image_base64":     "%s/sbcGetImgB64.php?%s",
	"get_nec_image":        "%s/sbcGetNecImg.php?%s",
	"get_nec_image_list":   "%s/sbcGetNecImgList.php?%s",
	"get_request_info":     "%s/sbcGetRequestInfo.php?%s",
	"get_small_image":      "%s/sbcGetSmlImg.php?%s",
	"get_small_image_list": "%s/sbcGetSmlImgList.php?%s",
	"text_to_speech":       "%s/sbcTextToSpeech.php?%s",
	"user_login":           "%s/sbcUserLogin.php?%s",
}

/*
The various image sizes. Presumably:
	M/S = Medium/Small resolution
	H/L = High/Low quality
There's also SS, but it's unscrambled and extremely low resoution and quality.
If you really want SS, use get_nec_image with regular filenames.
L has pretty bad artifacting, so most of the time S_H > M_L.
I've never seen anything over M, so for now I'm assuming it doesn't exist.
*/
var StaticImageSizes = []string{"M_H", "S_H", "M_L", "S_L"}

const staticImageUrlFmt = "%s/%s/%s.jpg"
const staticContentUrlFmt = "%s/content.js"

// The API doesn't always serve images over the API, but often redirects to a CDN.
type ContentServerType int

const (
	ServerTypeUnset  ContentServerType = iota - 1
	ServerTypeSbc                      // means the images should be downloaded through the provided sbc API
	ServerTypeStatic                   // means the images should be downloaded directly from the provided CDN
)

type ParamsGetter func(binb *Api, method string) map[string][]string

type Api struct {
	Bib, Cid, ContentServer string
	ContentInfo             *ContentInfoResponse
	Content                 *ContentResponse
	Descrambler             *Descrambler
	Pages, FullPages        []string
	K                       string
	ServerType              ContentServerType
	Session                 *http.Client
	Params                  ParamsGetter
}

type Response struct {
	Result int
	Items  []json.RawMessage
}

type ContentInfoResponse struct {
	Abstract string
	Authors  []struct {
		Name, Role, Ruby string
	}
	Categories                             []string
	ContentID, ContentType, ContentsServer string
	ServerType                             ContentServerType
	Title, TitleRuby                       string
	P, Ctbl, Ptbl, Atbl, Ttbl              string
}

type ContentResponse struct {
	ContentDate, ConverterType, ConverterVersion string
	SmlImageCnt, NecImageSize, NecImageCnt       int
	Ttx, Prop, SBCVersion                        string
}

func NewApi(bib, cid string, session *http.Client, params ParamsGetter) *Api {
	if session == nil {
		session = &http.Client{}
	}
	if params == nil {
		params = func(*Api, string) map[string][]string { return nil }
	}

	res := &Api{
		Bib:        strings.TrimSuffix(bib, "/"),
		Cid:        cid,
		ServerType: ServerTypeUnset,
		Session:    session,
		Params:     params,
		K:          generateK(),
	}
	res.Bib = strings.TrimSuffix(bib, "/")
	res.Cid = cid

	return res
}

// ====================================================================
//                             BIB METHODS
// ====================================================================

func (binb *Api) GetContentInfo() error {
	method := "get_content_info"
	params := url.Values{}
	params.Set("cid", binb.Cid)
	params.Set("k", binb.K)
	extraParams := binb.Params(binb, method)
	for k, v := range extraParams {
		params[k] = v
	}
	url := fmt.Sprintf(bibApi[method], binb.Bib, params.Encode())
	log.WithField("url", url).Debugf("Calling %s...", method)

	r, err := binb.Session.Get(url)
	if err != nil {
		return err
	} else if r.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP request returned error code: %d", r.StatusCode)
	}
	defer r.Body.Close()

	// Unmarshal into a Response struct.
	var res Response
	err = json.NewDecoder(r.Body).Decode(&res)
	if err != nil {
		return err
	}

	if res.Result != 1 {
		return fmt.Errorf("%s returned result: %d", method, res.Result)
	} else if len(res.Items) == 0 {
		return fmt.Errorf("%s returned an empty items list.", method)
	}

	// Unmarshal the rest of the data.
	var info ContentInfoResponse
	if err := json.Unmarshal(res.Items[0], &info); err != nil {
		return err
	}
	binb.ContentInfo = &info

	// Get encrypted scramble data if present and process it.
	// Get the decrypted bytes.
	cRaw, err := binb.decryptData(binb.ContentInfo.Ctbl)
	if err != nil {
		return err
	}
	pRaw, err := binb.decryptData(binb.ContentInfo.Ptbl)
	if err != nil {
		return err
	}

	// The decrypted bytes should be JSON, so we unmarshal them.
	var c, p []string
	if err := json.Unmarshal(cRaw, &c); err != nil {
		return err
	}
	if err := json.Unmarshal(pRaw, &p); err != nil {
		return err
	}

	// Create a descrambler with the decrypted data.
	binb.Descrambler, err = NewDescrambler(c, p)
	if err != nil {
		return err
	}

	// Get the content server.
	binb.ContentServer = strings.TrimSuffix(binb.ContentInfo.ContentsServer, "/")
	binb.ServerType = binb.ContentInfo.ServerType

	return nil
}

// ====================================================================
//                             SBC METHODS
// ====================================================================

func (binb *Api) GetContent() error {
	method := "get_content"
	if err := binb.ensureContentInfo(method); err != nil {
		return err
	}

	switch binb.ServerType {
	case ServerTypeSbc:
		if !binb.assertP() {
			return errors.New("Tried to use SBC without a p value set.")
		}
		// Start constructing the URL.
		params := url.Values{}
		params.Set("cid", binb.Cid)
		params.Set("p", binb.ContentInfo.P)
		extraParams := binb.Params(binb, method)
		for k, v := range extraParams {
			params[k] = v
		}
		url := fmt.Sprintf(sbcApi[method], binb.ContentServer, params.Encode())
		log.WithField("url", url).Debugf("Calling %s...", method)

		r, err := binb.Session.Get(url)
		if err != nil {
			return err
		}
		defer r.Body.Close()

		if r.StatusCode != http.StatusOK {
			return fmt.Errorf("HTTP request returned error code: %d", r.StatusCode)
		}

		s, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return err
		}
		var content ContentResponse
		if err := json.Unmarshal(s, &content); err != nil {
			return err
		}
		binb.Content = &content
		paths := reTtxImagePath.FindAllStringSubmatch(content.Ttx, -1)
		if paths == nil {
			return errors.New("No image listing found.")
		}
		// Populate the Pages and FullPages members.
		binb.Pages = make([]string, content.SmlImageCnt)
		binb.FullPages = make([]string, content.SmlImageCnt)
		for i := 0; i < content.SmlImageCnt; i++ {
			full := paths[i][1]
			binb.FullPages[i] = full
			// For Pages, only keep the base filename.
			binb.Pages[i] = full[strings.LastIndex(full, "/")+1:]
		}

		return nil
	case ServerTypeStatic:
		url := fmt.Sprintf(staticContentUrlFmt, binb.ContentServer)
		log.WithField("url", url).Debug("Getting content from CDN...")

		r, err := binb.Session.Get(url)
		if err != nil {
			return err
		} else if r.StatusCode != http.StatusOK {
			return fmt.Errorf("HTTP request returned error code: %d", r.StatusCode)
		}
		defer r.Body.Close()

		// Data is JS ran through eval().
		// Format: DataGet_Content(<JSON_GOES_HERE>)
		s, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return err
		} else if len(s) < 17 { // len("DataGet_Content()")
			return fmt.Errorf("content.js length shorter than expected: %d", len(s))
		}
		// Strip the JS and only leave the JSON.
		s = s[16 : len(s)-1]
		// Go ahead and unmarshal it.
		var content ContentResponse
		if err := json.Unmarshal(s, &content); err != nil {
			return err
		}
		binb.Content = &content
		paths := reTtxImagePath.FindAllStringSubmatch(content.Ttx, -1)
		if paths == nil {
			return errors.New("No image listing found.")
		}
		// Populate the Pages and FullPages members.
		binb.Pages = make([]string, content.SmlImageCnt)
		binb.FullPages = make([]string, content.SmlImageCnt)
		for i := 0; i < content.SmlImageCnt; i++ {
			full := paths[i][1]
			binb.FullPages[i] = full
			// For Pages, only keep the base filename.
			binb.Pages[i] = full[strings.LastIndex(full, "/")+1:]
		}

		return nil
	}

	return fmt.Errorf("Unknown content server type: %d", binb.ServerType)
}

func (binb *Api) GetImage(page int) (io.ReadCloser, error) {
	method := "get_image"
	if err := binb.ensureContent(method); err != nil {
		return nil, err
	}

	switch binb.ServerType {
	case ServerTypeSbc:
		// Start constructing the URL.
		params := url.Values{}
		params.Set("cid", binb.Cid)
		params.Set("p", binb.ContentInfo.P)
		params.Set("src", binb.FullPages[page])
		// Some parameters to make the API return the largest image.
		params.Set("h", "9999")
		params.Set("q", "0")
		extraParams := binb.Params(binb, method)
		for k, v := range extraParams {
			params[k] = v
		}
		url := fmt.Sprintf(sbcApi[method], binb.ContentServer, params.Encode())
		log.WithField("url", url).Debugf("Calling %s...", method)

		r, err := binb.Session.Get(url)
		if err != nil {
			return nil, err
		} else if r.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("HTTP request returned error code: %d", r.StatusCode)
		}

		return r.Body, nil
	case ServerTypeStatic:
		for _, size := range StaticImageSizes {
			url := fmt.Sprintf(staticImageUrlFmt, binb.ContentServer, binb.FullPages[page], size)
			log.WithField("url", url).Debug("Getting image from CDN...")

			r, err := binb.Session.Get(url)
			if err != nil {
				return nil, err
			} else if r.StatusCode == http.StatusNotFound {
				log.WithField("size", size).Debug("Image not found.")
				continue
			} else if r.StatusCode != http.StatusOK {
				// Some servers might return something other than 404 even if
				// the directory exists but perhaps not that particular image
				// size, so we do not return an error right away.
				log.Debugf("HTTP request returned error code: %d", r.StatusCode)
				continue
			}
			return r.Body, nil
		}

		// Tried all image sizes but never got an image.
		return nil, errors.New("Unable to get image from the CDN.")
	}

	return nil, fmt.Errorf("Unknown content server type: %d", binb.ServerType)
}

// ====================================================================
//                               HELPERS
// ====================================================================

func generateK() string {
	var res bytes.Buffer
	now := time.Now()
	source := fmt.Sprintf("%d%02d%02d%02d%02d%02d%d%s\n",
		now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(),
		now.Second(), now.Nanosecond()/1000, apiAlphabet)
	for i := 0; i < 32; i++ {
		res.WriteByte(source[int(rand.Float64()*float64(len(source)))])
	}

	return res.String()
}

func (binb *Api) decryptData(data string) ([]byte, error) {
	makeKey := func(cid, k string) uint32 {
		s := cid + ":" + k
		var res uint32
		for i, char := range s {
			res += uint32(char) << uint(i%16)
		}
		res &= 0x7FFFFFFF

		if res != 0 {
			return res
		} else {
			return 0x12345678
		}
	}

	key := makeKey(binb.Cid, binb.K)
	var buf bytes.Buffer
	for _, char := range data {
		key = (key >> 1) ^ (-(key & 1) & 0x48200004)
		if err := buf.WriteByte(byte(((uint32(char) - 0x20 + key) % 0x5E) + 0x20)); err != nil {
			panic(err)
		}
	}

	return buf.Bytes(), nil
}

func (binb *Api) ensureContentInfo(method string) error {
	if binb.ServerType == ServerTypeUnset {
		log.Debugf("%s called with an unset ServerType. Getting content info...", method)
		if err := binb.GetContentInfo(); err != nil {
			return fmt.Errorf("Failed to ensure content info: %s", err)
		}
	}

	return nil
}

func (binb *Api) ensureContent(method string) error {
	if err := binb.ensureContentInfo(method); err != nil {
		return err
	}

	if binb.Pages == nil {
		log.Debugf("%s called without the page list initialized. Getting content...", method)
		if err := binb.GetContent(); err != nil {
			return fmt.Errorf("Failed to ensure content: %s", err)
		}
	}

	return nil
}

func (binb *Api) assertP() bool {
	return binb.ContentInfo != nil && binb.ContentInfo.P != ""
}
