package bookwalker

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/url"
	"strconv"
	"time"

	"github.com/MinoMino/mindl/plugins"
	log "github.com/Sirupsen/logrus"
)

var (
	ErrBookWalkerFailedAuth    = errors.New("Failed to authenticate for a book session.")
	ErrBookWalkerFailedLogin   = errors.New("Failed to login. Wrong credentials?")
	ErrBookWalkerFailedLogout  = errors.New("Failed to logout. Did the API change?")
	ErrBookWalkerNoSession     = errors.New("Failed to get a book session.")
	ErrBookWalkerNoContent     = errors.New("Failed to get book content info.")
	ErrBookWalkerFailedContent = errors.New("Failed to process content info.")
	ErrBookWalkerNoConfig      = errors.New("Content info had no configuration key.")
)

func (bw *BookWalker) login(username, password string) {
	r, err := bw.client.Do(plugins.NewPostFormRequestUA(urlLogin, plugins.IE11UserAgent,
		url.Values{
			"j_username":      {username},
			"j_password":      {password},
			"j_platform_code": {"03"},
		}))
	if err != nil {
		log.Error(err)
		panic(ErrBookWalkerFailedLogin)
	}
	defer r.Body.Close()
	plugins.PanicForStatus(r, "Did the login API change?")

	// Confirm we logged in by checking the URL we got redirected to.
	if !reProfile.MatchString(r.Request.URL.String()) {
		panic(ErrBookWalkerFailedLogin)
	}
}

func (bw *BookWalker) logout() {
	r, err := bw.client.Do(plugins.NewGetRequestUA(urlLogout, plugins.IE11UserAgent))
	if err != nil {
		log.Error(err)
		panic(ErrBookWalkerFailedLogout)
	}
	defer r.Body.Close()
	plugins.PanicForStatus(r, "Did the logout API change?")
}

func (bw *BookWalker) getBookSession(cid string) (*BookSession, error) {
	bid := getBrowserId(browserIdSuffix)
	bookparams := url.Values{}
	bookparams.Set("cid", cid)
	authparams := url.Values{}
	authparams.Set("params", bookparams.Encode())
	authparams.Set("ref", "")
	myurl := urlApi + "/auth?" + authparams.Encode()
	log.WithField("url", myurl).Debug("Authenticating...")
	r, err := bw.client.Do(plugins.NewGetRequestUA(myurl, plugins.IE11UserAgent))
	if err != nil {
		log.Error(err)
		return nil, ErrBookWalkerFailedAuth
	}
	defer r.Body.Close()
	plugins.PanicForStatus(r, "Did the API change?")

	bookparams.Set("BID", bid)
	myurl = urlApi + "/c?" + bookparams.Encode()
	log.WithField("url", myurl).Debug("Getting book session...")
	r2, err := bw.client.Do(plugins.NewGetRequestUA(myurl, plugins.IE11UserAgent))
	if err != nil {
		log.Error(err)
		return nil, ErrBookWalkerNoSession
	}
	defer r2.Body.Close()
	plugins.PanicForStatus(r2, "Did the API change?")

	var res BookSession
	dec := json.NewDecoder(r2.Body)
	if err := dec.Decode(&res); err != nil {
		return nil, err
	}

	if res.Status != "200" {
		log.Errorf("Book session API returned status code: %s", res.Status)
		return &res, ErrBookWalkerNoSession
	}

	return &res, nil
}

func (bw *BookWalker) getContentInfo() (*BookConfig, []*BookContent, error) {
	params := url.Values{}
	params.Set("hti", bw.session.AuthInfo.Hti)
	params.Set("cfg", strconv.Itoa(bw.session.AuthInfo.Config))
	params.Set("Policy", bw.session.AuthInfo.Policy)
	params.Set("Signature", bw.session.AuthInfo.Signature)
	params.Set("Key-Pair-Id", bw.session.AuthInfo.KeyPairId)
	myurl := bw.session.Url + "configuration_pack.json" + "?" + params.Encode()
	log.WithField("url", myurl).Debug("Getting content info...")
	r, err := bw.client.Do(plugins.NewGetRequestUA(myurl, plugins.IE11UserAgent))
	if err != nil {
		return nil, nil, err
	}
	plugins.PanicForStatus(r, "Failed to get image.")
	defer r.Body.Close()

	dec := json.NewDecoder(r.Body)
	var res map[string]json.RawMessage
	if err := dec.Decode(&res); err != nil {
		log.Error(err)
		return nil, nil, ErrBookWalkerNoContent
	} else if _, ok := res["configuration"]; !ok {
		return nil, nil, ErrBookWalkerNoConfig
	}

	// Unmarshal the config.
	var config BookConfig
	if err := json.Unmarshal(res["configuration"], &config); err != nil {
		log.Error(err)
		return nil, nil, ErrBookWalkerNoContent
	}

	// Unmarshal the items.
	pages := make([]*BookContent, len(config.Contents))
	for i, c := range config.Contents {
		if err := json.Unmarshal(res[c.File], &pages[i]); err != nil {
			log.Error(err)
			return nil, nil, ErrBookWalkerFailedContent
		}
		pages[i].FilePath = c.File
	}

	return &config, pages, nil
}

func (bw *BookWalker) getImage(page, subpage int) (io.ReadCloser, error) {
	baseurl := bw.session.Url + bw.content[page-1].FilePath + "/" + strconv.Itoa(subpage) + ".jpeg"
	params := url.Values{}
	params.Set("hti", bw.session.AuthInfo.Hti)
	params.Set("cfg", strconv.Itoa(bw.session.AuthInfo.Config))
	params.Set("Policy", bw.session.AuthInfo.Policy)
	params.Set("Signature", bw.session.AuthInfo.Signature)
	params.Set("Key-Pair-Id", bw.session.AuthInfo.KeyPairId)
	myurl := baseurl + "?" + params.Encode()
	//log.WithField("url", myurl).Debug("Getting image...")
	r, err := bw.client.Do(plugins.NewGetRequestUA(myurl, plugins.IE11UserAgent))
	if err != nil {
		return nil, err
	}
	plugins.PanicForStatus(r, "Failed to get image.")

	return r.Body, nil
}

func getBrowserId(suffix string) string {
	r := int(rand.Float64() * 100000000)
	rs := fmt.Sprintf("%08d", r)
	now := time.Now().Unix() * 1000

	return fmt.Sprintf("%d%s%s", now, rs, suffix)
}
