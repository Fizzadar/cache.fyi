package content_archivers

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/fizzadar/cache.fyi/internal/config"
	"github.com/fizzadar/cache.fyi/internal/database"
	"github.com/fizzadar/cache.fyi/internal/types"
	"github.com/fizzadar/cache.fyi/internal/util"
	"github.com/rs/zerolog"
)

var _ ContentArchiver = (*LinkwardenArchiver)(nil)

type LinkwardenArchiver struct {
	config config.CachefyiConfig
	log    zerolog.Logger
	db     *database.Database
}

func NewLinkwardenArchiver(cfg config.CachefyiConfig, log zerolog.Logger, db *database.Database) *LinkwardenArchiver {
	return &LinkwardenArchiver{
		config: cfg,
		log:    log,
		db:     db,
	}
}

type linkwardenLinkResponse struct {
	Response struct {
		LastPreserved *time.Time `json:"lastPreserved"`
	} `json:"response"`
}

type archiveType struct {
	format int
	ext    string
}

// https://docs.linkwarden.app/api/retrieves-an-archive-file-by-link-id
var archiveTypes = []archiveType{
	{0, ".png"},
	{1, ".jpg"},
	{2, ".pdf"},
	{3, ".json"},
	{4, ".html"},
}

func (a *LinkwardenArchiver) ArchiveContent(ctx context.Context, c *types.Content) error {
	if c.LinkwardenLinkID == nil {
		if c.Type == types.ContentTypeURL {
			return fmt.Errorf("got url for archiving without linkwarden link id")
		}
		return nil
	}

	linkID := *c.LinkwardenLinkID

	for {
		req := util.LinkwardenRequest{
			APIToken: a.config.LinkwardenAPIToken,
			URL:      a.config.LinkwardenURL,
			Endpoint: fmt.Sprintf("/api/v1/links/%s", linkID),
		}

		var linkResp linkwardenLinkResponse
		resp, err := util.DoLinkwardenRequest(ctx, req)
		if err != nil {
			return fmt.Errorf("failed to check for link preserved")
		} else if err := json.Unmarshal(resp.Body, &linkResp); err != nil {
			return fmt.Errorf("failed to unmarshal link response")
		}

		if linkResp.Response.LastPreserved != nil {
			break
		}

		a.log.Debug().Msg("Link not yet preserved, waiting 10s...")
		time.Sleep(10 * time.Second)
	}

	for _, archiveType := range archiveTypes {
		log := a.log.With().
			Int("format", archiveType.format).
			Logger()

		req := util.LinkwardenRequest{
			APIToken: a.config.LinkwardenAPIToken,
			URL:      a.config.LinkwardenURL,
			Endpoint: fmt.Sprintf("/api/v1/archives/%s?format=%d", linkID, archiveType.format),
		}

		resp, err := util.DoLinkwardenRequest(ctx, req)
		if err != nil {
			log.Warn().Err(err).Msg("Linkwarden archive request failed")
			continue
		} else if len(resp.Body) == 0 {
			log.Warn().Msg("Linwarden archive empty response")
			continue
		}

		contentType := resp.Header.Get("Content-Type")
		if contentType == "" {
			log.Warn().Msg("Linkwarden archive has no content type")
			continue
		}

		filename := fmt.Sprintf("linkwarden_archive_%s%s", linkID, archiveType.ext)

		hashStr := string(types.ContentTypeFile) + c.URL + strconv.Itoa(archiveType.format)
		hashB := sha256.Sum256([]byte(hashStr))
		hash := base64.RawURLEncoding.EncodeToString(hashB[:])

		_, err = a.db.CreateContent(
			ctx,
			types.ContentTypeFile,
			hash,
			c.URL,
			resp.Body,
			&contentType,
			&filename,
			[]int64{},
			&c.ID,
		)
		if err != nil {
			return nil
		}

		log.Debug().
			Int("format", archiveType.format).
			Int("size", len(resp.Body)).
			Msg("Saved Linkwarden archive")
	}

	return nil
}
