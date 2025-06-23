package vrpaws

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"time"

	"github.com/disintegration/imaging"

	"vrc-moments/cmd/daemon/components/logger"
	lib "vrc-moments/pkg"
	"vrc-moments/pkg/api"
	"vrc-moments/pkg/vrc"
)

type uploadPayload struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	IsPublic    bool   `json:"isPublic"`

	StorageId          string `json:"storageId"`
	ThumbnailStorageId string `json:"thumbnailStorageId"`
	SmallStorageId     string `json:"smallStorageId"`
	MediumStorageId    string `json:"mediumStorageId"`
	LargeStorageId     string `json:"largeStorageId"`

	ClientTag string `json:"clientTag"`

	Metadata *Metadata `json:"vrcMetadata,omitempty"`
}

type Metadata struct {
	Author  vrc.User   `json:"author"`
	World   World      `json:"world"`
	Players []vrc.User `json:"players"`
}

type World struct {
	Name string `json:"name"`
	ID   string `json:"id"`
}

type UploadPayload struct {
	SetProgress func(logger.Renderable, float64)
	*api.UploadPayload
}

type UploadResponse struct {
	Image string `json:"image"`
}

func (s *Server) Upload(ctx context.Context, payload *UploadPayload) (*UploadResponse, error) {
	defer payload.File.Close()

	payload.SetProgress(logger.Message("Loading data..."), 0.05)
	if payload.Token == "" {
		return nil, errors.New("missing access token")
	}

	endpoint := *s.remote
	endpoint.Path = path.Join(endpoint.Path, "images", "upload")

	var main bytes.Buffer
	_, err := io.Copy(&main, payload.File.Data)
	if err != nil {
		return nil, fmt.Errorf("could not copy file to buffer: %w", err)
	}

	payload.SetProgress(logger.Message("Decoding image..."), 0.1)
	image, err := imaging.Decode(bytes.NewReader(main.Bytes()))
	if err != nil {
		return nil, fmt.Errorf("could not decode image: %w", err)
	}

	payload.SetProgress(logger.Message("Starting upload..."), 0.2)
	uploadPayload := uploadPayload{
		Title:       lib.RemoveExtension(payload.File.Filename),
		Description: "",
		IsPublic:    false,

		ClientTag: payload.File.SHA256,

		Metadata: metadata(payload.File.Metadata),
	}

	bounds := image.Bounds()

	numSteps := len(imageSizes)
	const stepStart = 0.2
	const stepEnd = 0.8
	stepIncrement := (stepEnd - stepStart) / float64(numSteps)
	var stepIndex int

	for mode, size := range imageSizes {
		var reader io.Reader

		// check if we exceed size.width or size.height
		if bounds.Dx() > size.width || bounds.Dy() > size.height {
			payload.SetProgress(logger.Messagef("Resizing %s...", mode), stepStart+(float64(stepIndex))*stepIncrement)
			reader, err = resize(image, size.width, size.height)
			if err != nil {
				return nil, fmt.Errorf("could not resize %s image: %w", mode, err)
			}
		} else {
			reader = bytes.NewReader(main.Bytes())
		}

		payload.SetProgress(logger.Messagef("Uploading %s...", mode), stepStart+(float64(stepIndex)+0.5)*stepIncrement)
		id, err := s.upload(payload.Token, reader)
		if err != nil {
			return nil, fmt.Errorf("could not upload %s image: %w", mode, err)
		}
		payload.SetProgress(logger.Messagef("Uploading %s...", mode), stepStart+(float64(stepIndex)+1)*stepIncrement)

		switch mode {
		case original:
			uploadPayload.StorageId = id
		case thumbnail:
			uploadPayload.ThumbnailStorageId = id
		case small:
			uploadPayload.SmallStorageId = id
		case medium:
			uploadPayload.MediumStorageId = id
		case large:
			uploadPayload.LargeStorageId = id
		}

		stepIndex++
	}

	reader, err := lib.Encode(uploadPayload)
	if err != nil {
		return nil, fmt.Errorf("could not encode upload payload: %w", err)
	}

	payload.SetProgress(logger.Message("Uploading..."), 0.85)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), reader)
	if err != nil {
		return nil, fmt.Errorf("could not create upload request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+payload.Token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not finalize upload: %w", err)
	}
	defer resp.Body.Close()

	payload.SetProgress(logger.Message("Reading response..."), 0.95)
	if resp.StatusCode != http.StatusOK {
		bin, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code: %s: %s", resp.Status, bin)
	}

	response, err := lib.Decode[*UploadResponse](resp.Body)
	if err != nil {
		return nil, fmt.Errorf("could not decode upload response: %w", err)
	}

	payload.SetProgress(logger.Message("Done!"), 1.0)
	return response, nil
}

type storageID struct {
	StorageID string `json:"storageID"`
}

type uploadURL struct {
	URL string `json:"token"`
}

func (s *Server) upload(accessToken string, file io.Reader) (string, error) {
	endpoint, err := s.getUploadToken(accessToken)
	if err != nil {
		return "", fmt.Errorf("could not get upload token: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, endpoint, file)
	if err != nil {
		return "", fmt.Errorf("could not create upload request: %w", err)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("could not send upload request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("upload request failed with status %s", resp.Status)
	}

	id, err := lib.Decode[storageID](resp.Body)
	if err != nil {
		return "", fmt.Errorf("could not decode upload response: %w", err)
	}

	return id.StorageID, nil
}

func (s *Server) getUploadToken(accessToken string) (string, error) {
	ctx, done := context.WithTimeout(s.context, 30*time.Second)
	defer done()

	u := *s.remote
	u.Path = path.Join(u.Path, "images", "get-upload-token")

	q := u.Query()
	q.Add("accessToken", accessToken)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("get token request failed with status %s", resp.Status)
	}

	token, err := lib.Decode[uploadURL](resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to decode token: %w", err)
	}

	return token.URL, nil
}
