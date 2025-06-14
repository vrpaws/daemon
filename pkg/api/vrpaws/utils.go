package vrpaws

import (
	"bytes"
	"fmt"
	"image"
	"io"

	"github.com/HugoSmits86/nativewebp"
	"github.com/disintegration/imaging"

	"vrc-moments/pkg/vrc"
)

type imageSize struct {
	width  int
	height int
}

const (
	original  = "original"
	thumbnail = "thumbnail"
	small     = "small"
	medium    = "medium"
	large     = "large"
)

var imageSizes = map[string]imageSize{
	original:  {width: 3840, height: 3840},
	thumbnail: {width: 250, height: 250},
	small:     {width: 600, height: 600},
	medium:    {width: 1200, height: 1200},
	large:     {width: 2000, height: 2000},
}

func resize(img image.Image, width, height int) (io.Reader, error) {
	img = imaging.Fit(img, width, height, imaging.Lanczos)
	var buf bytes.Buffer
	err := nativewebp.Encode(&buf, img, nil)
	if err != nil {
		return nil, fmt.Errorf("could not encode image: %w", err)
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
