package vrpaws

import (
	"bytes"
	"fmt"
	"image"

	"github.com/disintegration/imaging"
	"github.com/gen2brain/webp"

	"vrc-moments/pkg/vrc"
)

type imageParam struct {
	width  int
	height int

	quality int
}

const (
	original  = "original"
	thumbnail = "thumbnail"
	small     = "small"
	medium    = "medium"
	large     = "large"
)

var imageParams = map[string]imageParam{
	original:  {width: 3840, height: 3840, quality: 95},
	thumbnail: {width: 250, height: 250, quality: 80},
	small:     {width: 600, height: 600, quality: 80},
	medium:    {width: 1200, height: 1200, quality: 80},
	large:     {width: 2000, height: 2000, quality: 95},
}

func resize(img image.Image, param imageParam) (*bytes.Buffer, error) {
	img = imaging.Fit(img, param.width, param.height, imaging.Lanczos)
	var buf bytes.Buffer
	opts := webp.Options{
		Quality:  param.quality,
		Method:   6,
		Lossless: false,
	}
	if err := webp.Encode(&buf, img, opts); err != nil {
		return nil, fmt.Errorf("could not encode webp: %w", err)
	}
	return &buf, nil
}

func metadata(metadata *vrc.Metadata) *Metadata {
	if metadata == nil {
		return nil
	}

	return &Metadata{
		Author: metadata.Author,
		World: World{
			Name: metadata.World.Name,
			ID:   metadata.World.ID,
		},
		Players: metadata.Players,
	}
}
