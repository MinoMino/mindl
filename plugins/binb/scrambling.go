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
	"errors"
	"image"
	"io"
	"regexp"
	"strconv"
	"strings"

	_ "image/jpeg"
	_ "image/png"

	"github.com/MinoMino/mindl/logger"
)

var reType1Key = regexp.MustCompile("^=([0-9]+)-([0-9]+)([-+])([0-9]+)-([-_0-9A-Za-z]+)$")
var reType2Key = regexp.MustCompile("^([0-9]+?)-([0-9]+?)-([A-Za-z]+)$")

var tnpConstants = [...]int{
	-1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1,
	-1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1,
	-1, -1, -1, -1, -1, -1, -1, 62, -1, -1, 52, 53, 54, 55, 56, 57, 58, 59, 60,
	61, -1, -1, -1, -1, -1, -1, -1, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12,
	13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, -1, -1, -1, -1, 63, -1,
	26, 27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44,
	45, 46, 47, 48, 49, 50, 51, -1, -1, -1, -1, -1,
}

type scrambleKeyType int

const (
	typeUnset scrambleKeyType = iota + 1
	type1
	type2
)

const scrambleAlphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"

// A single rectangle in a scrambled image.
type scrambleRectangle struct {
	src, dst      image.Point
	width, height int
}

// All the rectangles for a scrambled image.
type scrambleRectanglesCollection struct {
	rectangles          []*scrambleRectangle
	srcWidth, srcHeight int
	dstWidth, dstHeight int
}

type scrambleDataType1 struct {
	h, v, padding int
	src, dst      string
}

type scrambleDataType2Piece struct {
	pos           image.Point
	width, height int
}

type scrambleDataType2 struct {
	ndx, ndy int
	pieces   []*scrambleDataType2Piece
}

type scrambleDataType2Pair struct {
	c, p *scrambleDataType2
}

type Descrambler struct {
	Ctbl, Ptbl           []string
	keyType              scrambleKeyType
	data                 []interface{}
	rectangleCollections [][]*scrambleRectanglesCollection
}

func NewDescrambler(ctbl, ptbl []string) (*Descrambler, error) {
	if len(ctbl) != len(ptbl) {
		log.WithFields(logger.Fields{
			"ctbl": ctbl,
			"ptbl": ptbl,
		}).Debug("ctbl and ptbl sizes don't match.")
		return nil, errors.New("ctbl and ptbl need to be of the same size.")
	} else if len(ctbl) == 0 {
		return nil, errors.New("ctbl cannot be empty.")
	} else if len(ptbl) == 0 {
		return nil, errors.New("ptbl cannot be empty.")
	}

	res := &Descrambler{
		Ctbl:                 ctbl,
		Ptbl:                 ptbl,
		keyType:              typeUnset,
		data:                 make([]interface{}, len(ctbl)),
		rectangleCollections: make([][]*scrambleRectanglesCollection, len(ctbl)),
	}
	for i := 0; i < len(res.rectangleCollections); i++ {
		res.rectangleCollections[i] = make([]*scrambleRectanglesCollection, len(ptbl))
	}

	err := res.init()
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (ds *Descrambler) init() error {
	var newType scrambleKeyType
	var data interface{}
	var err error

	for i := 0; i < len(ds.Ctbl); i++ {
		if ds.Ctbl[i][0] == '=' && ds.Ptbl[i][0] == '=' {
			newType = type1
			data, err = ds.processType1(i)
		} else if startsWithDigit(ds.Ctbl[i]) && startsWithDigit(ds.Ptbl[i]) {
			var c, p *scrambleDataType2
			newType = type2
			c, err = ds.processType2(ds.Ctbl[i])
			if err == nil {
				p, err = ds.processType2(ds.Ptbl[i])
			}
			if err == nil {
				data = &scrambleDataType2Pair{
					c: c,
					p: p,
				}
			}
		} else {
			log.WithFields(logger.Fields{
				"ctbl": ds.Ctbl[i],
				"ptbl": ds.Ptbl[i],
			}).Debug("Got unknown key type.")
			return errors.New("Unknown key type.")
		}

		if ds.keyType == typeUnset {
			ds.keyType = newType
		} else if ds.keyType != newType {
			log.WithFields(logger.Fields{
				"ctbl":      ds.Ctbl[i],
				"ptbl":      ds.Ptbl[i],
				"last_type": ds.keyType,
			}).Debugf("Got mixed key types while parsing type %d.", newType)
			return errors.New("Mixed key types.")
		}

		if err != nil {
			return err
		} else {
			ds.data[i] = data
		}
	}

	return nil
}

func (ds *Descrambler) processType1(i int) (*scrambleDataType1, error) {
	c := reType1Key.FindStringSubmatch(ds.Ctbl[i])
	p := reType1Key.FindStringSubmatch(ds.Ptbl[i])
	if c == nil || p == nil || c[1] != p[1] || c[2] != p[2] ||
		c[4] != p[4] || c[3] != "+" || p[3] != "-" {
		log.WithFields(logger.Fields{
			"ctbl": ds.Ctbl[i],
			"ptbl": ds.Ptbl[i],
		}).Debug("Type 1 key verification failed.")
		return nil, errors.New("Invalid type 1 scramble key.")
	}

	h, _ := strconv.Atoi(c[1])
	v, _ := strconv.Atoi(c[2])
	padding, _ := strconv.Atoi(c[4])
	if h > 8 || v > 8 || h*v > 64 {
		log.WithFields(logger.Fields{
			"h": h,
			"v": v,
		}).Debug("Invalid h and v values.")
		return nil, errors.New("Invalid h and v values.")
	}

	src := c[5]
	dst := p[5]
	target_len := h + v + h*v
	if len(src) != target_len || len(dst) != target_len {
		log.WithFields(logger.Fields{
			"h": h,
			"v": v,
		}).Debug("h and v do not match target length.")
		return nil, errors.New("Invalid h or v length.")
	}

	return &scrambleDataType1{h, v, padding, src, dst}, nil
}

func (ds *Descrambler) processType2(key string) (*scrambleDataType2, error) {
	decodeType2Key := func(char byte) int {
		c := strings.IndexByte(scrambleAlphabet, char)
		if c == -1 {
			return strings.IndexByte(strings.ToLower(scrambleAlphabet), char) * 2
		} else {
			return 1 + c*2
		}
	}

	re := reType2Key.FindStringSubmatch(key)
	if re == nil {
		log.WithField("key", key).Debug("Invalid format of a type 2 key.")
		return nil, errors.New("Invalid format of a type 2 key.")
	}

	ndx, _ := strconv.Atoi(re[1])
	ndy, _ := strconv.Atoi(re[2])
	data := re[3]
	if len(data) != ndx*ndy*2 {
		log.WithFields(logger.Fields{
			"ndx":  ndx,
			"ndy":  ndy,
			"data": data,
		}).Debug("Failed to validate type 2 data with ndx and ndy.")
		return nil, errors.New("Invalid key. Key data length does not match the rest.")
	}

	f := (ndx-1)*(ndy-1) - 1
	g := f + (ndx - 1)
	h := g + (ndy - 1)
	j := h + 1
	pieces := make([]*scrambleDataType2Piece, ndx*ndy)
	for i := 0; i < len(pieces); i++ {
		piece := &scrambleDataType2Piece{
			pos: image.Point{
				X: decodeType2Key(data[i*2]),
				Y: decodeType2Key(data[i*2+1]),
			},
		}
		if i <= f {
			piece.width = 2
			piece.height = 2
		} else if i <= g {
			piece.width = 2
			piece.height = 1
		} else if i <= h {
			piece.width = 1
			piece.height = 2
		} else if i <= j {
			piece.width = 1
			piece.height = 1
		}
		pieces[i] = piece
	}

	return &scrambleDataType2{
		ndx:    ndx,
		ndy:    ndy,
		pieces: pieces,
	}, nil
}

func (ds *Descrambler) rectanglesType1(cIndex, pIndex, srcWidth, srcHeight int) (*scrambleRectanglesCollection, error) {
	cData := ds.data[cIndex].(*scrambleDataType1)
	pData := ds.data[pIndex].(*scrambleDataType1)
	h := cData.h
	v := pData.v
	padding := cData.padding

	x := h * 2 * padding
	y := v * 2 * padding
	var width, height int
	if srcWidth >= 64+x && srcHeight >= 64+y && srcWidth*srcHeight >= (320+x)*(320+y) {
		width = srcWidth - h*2*padding
		height = srcHeight - v*2*padding
	} else {
		width = srcWidth
		height = srcHeight
	}

	srcT, srcN, srcP := tnp(cData.src, h, v)
	dstT, dstN, dstP := tnp(pData.dst, h, v)
	p := make([]int, h*v)
	for i := 0; i < h*v; i++ {
		p[i] = srcP[dstP[i]]
	}

	sliceWidth := (width + h - 1) / h
	sliceHeight := (height + v - 1) / v
	lastSliceWidth := width - (h-1)*sliceWidth
	lastSliceHeight := height - (v-1)*sliceHeight

	res := make([]*scrambleRectangle, h*v)
	for i := 0; i < len(res); i++ {
		dstColumn := i % h
		dstRow := i / h

		var dstX, dstY int
		dstX = padding + dstColumn*(sliceWidth+2*padding)
		if dstN[dstRow] < dstColumn {
			dstX += lastSliceWidth - sliceWidth
		}
		dstY = padding + dstRow*(sliceHeight+2*padding)
		if dstT[dstColumn] < dstRow {
			dstY += lastSliceHeight - sliceHeight
		}

		srcColumn := p[i] % h
		srcRow := p[i] / h

		var srcX, srcY int
		srcX = srcColumn * sliceWidth
		if srcN[srcRow] < srcColumn {
			srcX += lastSliceWidth - sliceWidth
		}
		srcY = srcRow * sliceHeight
		if srcT[srcColumn] < srcRow {
			srcY += lastSliceHeight - sliceHeight
		}

		var pWidth, pHeight int
		if dstN[dstRow] == dstColumn {
			pWidth = lastSliceWidth
		} else {
			pWidth = sliceWidth
		}
		if dstT[dstColumn] == dstRow {
			pHeight = lastSliceHeight
		} else {
			pHeight = sliceHeight
		}
		// For whatever reason, dst and src (called just d and s in the JS) switch places here.
		res[i] = &scrambleRectangle{
			dst:    image.Point{srcX, srcY},
			src:    image.Point{dstX, dstY},
			width:  pWidth,
			height: pHeight,
		}
	}

	return &scrambleRectanglesCollection{
		srcWidth:   srcWidth,
		srcHeight:  srcHeight,
		dstWidth:   width,
		dstHeight:  height,
		rectangles: res,
	}, nil
}

func (ds *Descrambler) rectanglesType2(cIndex, pIndex, srcWidth, srcHeight int) (*scrambleRectanglesCollection, error) {
	if !(srcWidth >= 64 && srcHeight >= 64 && srcWidth*srcHeight >= 320*320) {
		log.WithFields(logger.Fields{
			"srcWidth":  srcWidth,
			"srcHeight": srcHeight,
		}).Debug("Invalid input image dimensions.")
		return nil, errors.New("Invalid input image dimensions.")
	}

	e := srcWidth - (srcWidth % 8)
	f := ((e - 1) / 7) - ((e-1)/7)%8
	g := e - f*7
	h := srcHeight - (srcHeight % 8)
	j := ((h - 1) / 7) - ((h-1)/7)%8
	k := h - j*7

	pair := ds.data[cIndex].(*scrambleDataType2Pair)
	cData := pair.c
	pair = ds.data[pIndex].(*scrambleDataType2Pair)
	pData := pair.p
	res := make([]*scrambleRectangle, len(cData.pieces))
	for i := 0; i < len(res); i++ {
		cPiece := cData.pieces[i]
		pPiece := pData.pieces[i]
		srcX := (cPiece.pos.X/2)*f + (cPiece.pos.X%2)*g
		srcY := (cPiece.pos.Y/2)*j + (cPiece.pos.Y%2)*k
		dstX := (pPiece.pos.X/2)*f + (pPiece.pos.X%2)*g
		dstY := (pPiece.pos.Y/2)*j + (pPiece.pos.Y%2)*k
		width := (cPiece.width/2)*f + (cPiece.width%2)*g
		height := (cPiece.height/2)*j + (cPiece.height%2)*k
		res[i] = &scrambleRectangle{
			src:    image.Point{srcX, srcY},
			dst:    image.Point{dstX, dstY},
			width:  width,
			height: height,
		}
	}

	e = f*(cData.ndx-1) + g
	h = j*(cData.ndy-1) + k
	if e < srcWidth {
		res = append(res, &scrambleRectangle{
			src:    image.Point{e, 0},
			dst:    image.Point{e, 0},
			width:  srcWidth - e,
			height: h,
		})
	}
	if h < srcHeight {
		res = append(res, &scrambleRectangle{
			src:    image.Point{0, h},
			dst:    image.Point{0, h},
			width:  srcWidth,
			height: srcHeight - h,
		})
	}

	return &scrambleRectanglesCollection{
		srcWidth:   srcWidth,
		srcHeight:  srcHeight,
		dstWidth:   srcWidth,
		dstHeight:  srcHeight,
		rectangles: res,
	}, nil
}

func (ds *Descrambler) Descramble(filename string, reader io.Reader) (image.Image, error) {
	img, _, err := image.Decode(reader)
	if err != nil {
		return nil, err
	}

	bounds := img.Bounds()
	srcWidth := bounds.Dx()
	srcHeight := bounds.Dy()

	c, p := cpIndex(filename)

	/*
		If we've previously calculated the rectangles for these indices and the
		source image resolution hasn't changed, we'll reuse it. Otherwise calculate
		the rectangles and save them for potential future use.

		All this makes the code quite a bit more convoluted, but we'll often find
		ourselves descrambling ~200 images of the same resolution with usually a max
		of 64 different combinations of rectangles, so it's probably worth the trouble.
	*/
	col := &ds.rectangleCollections[c][p]
	if *col == nil || (*col != nil && (srcWidth != (*col).srcWidth || srcHeight != (*col).srcHeight)) {
		switch ds.keyType {
		case type1:
			*col, err = ds.rectanglesType1(c, p, srcWidth, srcHeight)
		case type2:
			*col, err = ds.rectanglesType2(c, p, srcWidth, srcHeight)
		default:
			log.WithField("type", ds.keyType).Debug("Found unknown key type while descrambling.")
			return nil, errors.New("Tried to descramble with unknown key type.")
		}
	}

	if err != nil {
		return nil, err
	}

	res := image.NewRGBA(image.Rect(0, 0, (*col).dstWidth, (*col).dstHeight))
	for _, rect := range (*col).rectangles {
		for x := 0; x < rect.width; x++ {
			for y := 0; y < rect.height; y++ {
				res.Set(x+rect.dst.X, y+rect.dst.Y, img.At(x+rect.src.X, y+rect.src.Y))
			}
		}
	}

	return res, nil
}

// Helpers.

func tnp(data string, h, v int) ([]int, []int, []int) {
	t := make([]int, h)
	n := make([]int, v)
	p := make([]int, h*v)

	for i := 0; i < h; i++ {
		t[i] = tnpConstants[data[i]]
	}
	for i := 0; i < v; i++ {
		n[i] = tnpConstants[data[h+i]]
	}
	for i := 0; i < h*v; i++ {
		p[i] = tnpConstants[data[h+v+i]]
	}

	return t, n, p
}

func cpIndex(filename string) (int, int) {
	c := 0
	p := 0
	for i, chr := range filename {
		if i%2 == 0 {
			p += int(chr)
		} else {
			c += int(chr)
		}
	}
	c %= 8
	p %= 8

	return c, p
}

func startsWithDigit(s string) bool {
	return '0' <= s[0] && s[0] <= '9'
}
