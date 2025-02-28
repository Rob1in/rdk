package rimage

import (
	"context"
	"image"
	"image/color"
	"sync"
)

// LazyEncodedImage defers the decoding of an image until necessary.
type LazyEncodedImage struct {
	imgBytes []byte
	mimeType string

	decodeOnce   sync.Once
	decodeErr    interface{}
	decodedImage image.Image
}

// NewLazyEncodedImage returns a new image that will only get decoded once actual data is needed
// from it. This is helpful for zero copy scenarios. If a width or height of the image is unknown,
// pass 0 or -1; when done a decode will happen on Bounds. In the future this can probably go
// away with reading all metadata from the header of the image bytes.
// NOTE: Usage of an image that would fail to decode causes a lazy panic.
func NewLazyEncodedImage(imgBytes []byte, mimeType string) image.Image {
	return &LazyEncodedImage{
		imgBytes: imgBytes,
		mimeType: mimeType,
	}
}

func (lei *LazyEncodedImage) decode() {
	lei.decodeOnce.Do(func() {
		defer func() {
			if err := recover(); err != nil {
				lei.decodeErr = err
			}
		}()
		lei.decodedImage, lei.decodeErr = DecodeImage(
			context.Background(),
			lei.imgBytes,
			lei.mimeType,
		)
	})
	if lei.decodeErr != nil {
		panic(lei.decodeErr)
	}
}

// MIMEType returns the encoded Image's MIME type.
func (lei *LazyEncodedImage) MIMEType() string {
	return lei.mimeType
}

// RawData returns the encoded Image's raw data.
// Note: This is not a copy and should only be read from.
func (lei *LazyEncodedImage) RawData() []byte {
	return lei.imgBytes
}

// ColorModel returns the Image's color model.
func (lei *LazyEncodedImage) ColorModel() color.Model {
	lei.decode()
	return lei.decodedImage.ColorModel()
}

// Bounds returns the domain for which At can return non-zero color.
// The bounds do not necessarily contain the point (0, 0).
func (lei *LazyEncodedImage) Bounds() image.Rectangle {
	if lei.decodedImage == nil {
		lei.decode()
	}
	return lei.decodedImage.Bounds()
}

// At returns the color of the pixel at (x, y).
// At(Bounds().Min.X, Bounds().Min.Y) returns the upper-left pixel of the grid.
// At(Bounds().Max.X-1, Bounds().Max.Y-1) returns the lower-right one.
func (lei *LazyEncodedImage) At(x, y int) color.Color {
	lei.decode()
	return lei.decodedImage.At(x, y)
}
