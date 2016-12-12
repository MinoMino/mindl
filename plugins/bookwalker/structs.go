package bookwalker

type BookSession struct {
	Status   string `json:"status"`
	Url      string `json:"url"`
	Title    string `json:"cti"`
	Lp       int    `json:"lp"`
	Cty      int    `json:"cty"`
	Lin      int    `json:"lin"`
	Lpd      int    `json:"lpd"`
	Bs       int    `json:"bs"`
	AuthInfo struct {
		Hti       string `json:"hti"`
		Config    int    `json:"cfg"`
		Policy    string `json:"Policy"`
		Signature string `json:"Signature"`
		KeyPairId string `json:"Key-Pair-Id"`
	} `json:"auth_info"`
	Tri string `json:"tri"`
}

type BookConfig struct {
	Contents []struct {
		File             string `json:"file"`
		Index            int    `json:"index"`
		OriginalFilePath string `json:"original-file-path"`
		Type             string `json:"type"`
	} `json:"contents"`
	JSONFormatVersion string `json:"json-format-version"`
	NavLists          []struct {
		Heading string `json:"heading"`
		Items   []struct {
			Hidden bool          `json:"hidden"`
			Href   string        `json:"href"`
			Label  string        `json:"label"`
			Nest   int           `json:"nest"`
			Types  []interface{} `json:"types"`
		} `json:"items"`
		Types []string `json:"types"`
	} `json:"nav-lists"`
	BookmarkSymbol           string      `json:"bookmarkSymbol"`
	PageProgressionDirection string      `json:"page-progression-direction"`
	PrerendererVersion       interface{} `json:"prerenderer-version"`
	TocList                  []struct {
		Hidden bool          `json:"hidden"`
		Href   string        `json:"href"`
		Label  string        `json:"label"`
		Nest   int           `json:"nest"`
		Types  []interface{} `json:"types"`
	} `json:"toc-list"`
}

type BookContent struct {
	FilePath               string      // Custom.
	BookmarkPositionToPage interface{} `json:"BookmarkPositionToPage"`
	FileLinkInfo           struct {
		PageCount        int `json:"PageCount"`
		PageLinkInfoList []struct {
			Page struct {
				ContentArea struct {
					Height int `json:"Height"`
					Width  int `json:"Width"`
					X      int `json:"X"`
					Y      int `json:"Y"`
				} `json:"ContentArea"`
				LinkList []interface{} `json:"LinkList"`
				No       int           `json:"No"`
				Rect     struct {
					Height int `json:"Height"`
					Width  int `json:"Width"`
					X      int `json:"X"`
					Y      int `json:"Y"`
				} `json:"Rect"`
				Shrink float64 `json:"Shrink"`
				Size   struct {
					Height int `json:"Height"`
					Width  int `json:"Width"`
				} `json:"Size"`
				DummyWidth  int `json:"DummyWidth"`
				DummyHeight int `json:"DummyHeight"`
			} `json:"Page"`
		} `json:"PageLinkInfoList"`
	} `json:"FileLinkInfo"`
	FixedLayoutSpec struct {
		AccessOrientation int `json:"AccessOrientation"`
		AccessScroll      int `json:"AccessScroll"`
		DeviceOrientation int `json:"DeviceOrientation"`
		PageSide          int `json:"PageSide"`
		RenditionLayout   int `json:"RenditionLayout"`
		RenditionSpread   int `json:"RenditionSpread"`
	} `json:"FixedLayoutSpec"`
	IDToPage       interface{} `json:"IdToPage"`
	MarginOff      int         `json:"MarginOff"`
	PageToBookmark interface{} `json:"PageToBookmark"`
	Title          string      `json:"Title"`
}
