package bookwalker

import (
	"image"
	"io"
	"sync"
)

const (
	rectangleWidth  = 64
	rectangleHeight = 64
	patternCount    = 4

	// A bunch of constants that aren't immediately obvious as to
	// what they should be called, so I'm just keeping the same name
	// as the JS, with a prefix.

	jsa = 61
	jsb = 73
	jsc = 4
	jsd = 43
	jse = 47
	jsf = 53
	jsg = 59
	jsh = 67
	jsi = 71
	jsj = 29
	jsk = 37
	jsl = 31
	jsm = 41
)

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

// Return the pattern that should be used for that particular filename.
func getPattern(filePath string) int {
	res := 0
	for _, c := range filePath {
		res += int(c)
	}

	return res%patternCount + 1
}

// Generate rectangles for the specific pattern and image size.
// We apparently have different notions as to what is the
// source and destination image, so src and dst are the
// other way around from the JS code.
func generateRectangles(srcWidth, srcHeight, pattern int) []*scrambleRectangle {
	rectsX := srcWidth / rectangleWidth
	rectsY := srcHeight / rectangleHeight
	remainderX := srcWidth % rectangleWidth
	remainderY := srcHeight % rectangleHeight
	res := make([]*scrambleRectangle, 0, rectsX*rectsY)

	i := rectsX - jsd*pattern%rectsX
	if i%rectsX == 0 {
		i = (rectsX - jsc) % rectsX
	}
	if i == 0 {
		i = rectsX - 1
	}

	n := rectsY - jse*pattern%rectsY
	if n%rectsY == 0 {
		n = (rectsY - jsc) % rectsY
	}
	if n == 0 {
		n = rectsY - 1
	}

	if remainderX > 0 && remainderY > 0 {
		o := i * rectangleWidth
		p := n * rectangleHeight

		res = append(res, &scrambleRectangle{
			dst:    image.Point{o, p},
			src:    image.Point{o, p},
			width:  remainderX,
			height: remainderY,
		})
	}

	if remainderY > 0 {
		for s := 0; s < rectsX; s++ {
			u := calcXCoordinateXRest(s, rectsX, pattern)
			v := calcYCoordinateXRest(u, i, n, rectsY, pattern)
			q := calcPositionWithRest(u, i, remainderX, rectangleWidth)
			r := v * rectangleHeight
			o := calcPositionWithRest(s, i, remainderX, rectangleWidth)
			p := n * rectangleHeight

			res = append(res, &scrambleRectangle{
				dst:    image.Point{o, p},
				src:    image.Point{q, r},
				width:  rectangleWidth,
				height: remainderY,
			})
		}
	}

	if remainderX > 0 {
		for t := 0; t < rectsY; t++ {
			v := calcYCoordinateYRest(t, rectsY, pattern)
			u := calcXCoordinateYRest(v, i, n, rectsX, pattern)
			q := u * rectangleWidth
			r := calcPositionWithRest(v, n, remainderY, rectangleHeight)
			o := i * rectangleWidth
			p := calcPositionWithRest(t, n, remainderY, rectangleHeight)

			res = append(res, &scrambleRectangle{
				dst:    image.Point{o, p},
				src:    image.Point{q, r},
				width:  remainderX,
				height: rectangleHeight,
			})
		}
	}

	for s := 0; s < rectsX; s++ {
		for t := 0; t < rectsY; t++ {
			u := (s + pattern*jsj + jsl*t) % rectsX
			v := (t + pattern*jsk + jsm*u) % rectsY
			w := 0
			if u >= calcXCoordinateYRest(v, i, n, rectsX, pattern) {
				w = remainderX
			}
			x := 0
			if v >= calcYCoordinateXRest(u, i, n, rectsY, pattern) {
				x = remainderY
			}
			q := u*rectangleWidth + w
			r := v*rectangleHeight + x
			o := s * rectangleWidth
			if s >= i {
				o += remainderX
			}
			p := t * rectangleHeight
			if t >= n {
				p += remainderY
			}

			res = append(res, &scrambleRectangle{
				dst:    image.Point{o, p},
				src:    image.Point{q, r},
				width:  rectangleWidth,
				height: rectangleHeight,
			})
		}
	}

	return res
}

// Some helpers used in the JS code.

func calcPositionWithRest(coordX, i, remainderX, rectangleSize int) int {
	res := coordX * rectangleSize
	if res >= i {
		return res + remainderX
	}

	return res
}

func calcXCoordinateXRest(index, rectangleCountX, pattern int) int {
	return (index + jsa*pattern) % rectangleCountX
}

func calcYCoordinateXRest(coordX, unk1, unk2, rectangleCountY, pattern int) int {
	l := pattern%2 == 1
	var k bool
	if coordX < unk1 {
		k = l
	} else {
		k = !l
	}

	var modulo, extra int
	if k {
		modulo = unk2
		extra = 0
	} else {
		modulo = rectangleCountY - unk2
		extra = unk2
	}

	return (coordX+pattern*jsf+unk2*jsg)%modulo + extra
}

func calcXCoordinateYRest(coordY, unk1, unk2, rectangleCountX, pattern int) int {
	l := pattern%2 == 1
	var k bool
	if coordY < unk2 {
		k = l
	} else {
		k = !l
	}

	var modulo, extra int
	if k {
		modulo = rectangleCountX - unk1
		extra = unk1
	} else {
		modulo = unk1
		extra = 0
	}

	return (coordY+pattern*jsh+unk1+jsi)%modulo + extra
}

func calcYCoordinateYRest(index, rectangleCountY, pattern int) int {
	return (index + jsb*pattern) % rectangleCountY
}

// Descrambler. Not from the JS code.
type descrambler struct {
	rectangleCollections [patternCount]*scrambleRectanglesCollection
	m                    sync.Mutex
}

func (ds *descrambler) Descramble(filename string, reader io.Reader, dummyWidth, dummyHeight int) (image.Image, error) {
	img, _, err := image.Decode(reader)
	if err != nil {
		return nil, err
	}

	bounds := img.Bounds()
	srcWidth := bounds.Dx()
	srcHeight := bounds.Dy()

	pattern := getPattern(filename)

	/*
	   If we've previously calculated the rectangles for this pattern and the
	   source image resolution hasn't changed, we'll reuse it. Otherwise calculate
	   the rectangles and save them for potential future use.
	*/
	ds.m.Lock()
	col := ds.rectangleCollections[pattern-1]
	if col == nil || srcWidth != col.srcWidth || srcHeight != col.srcHeight {
		// Generate the rectangles.
		ds.rectangleCollections[pattern-1] = &scrambleRectanglesCollection{
			rectangles: generateRectangles(srcWidth, srcHeight, pattern),
			srcWidth:   srcWidth,
			srcHeight:  srcHeight,
			dstWidth:   srcWidth - dummyWidth,
			dstHeight:  srcHeight - dummyHeight,
		}
		col = ds.rectangleCollections[pattern-1]
	}
	ds.m.Unlock()

	res := image.NewRGBA(image.Rect(0, 0, col.dstWidth, col.dstHeight))
	for _, rect := range col.rectangles {
		for x := 0; x < rect.width; x++ {
			for y := 0; y < rect.height; y++ {
				res.Set(x+rect.dst.X, y+rect.dst.Y, img.At(x+rect.src.X, y+rect.src.Y))
			}
		}
	}

	return res, nil
}
