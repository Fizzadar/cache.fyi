package content_processors

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/fizzadar/cache.fyi/internal/config"
	"github.com/fizzadar/cache.fyi/internal/database"
	"github.com/fizzadar/cache.fyi/internal/types"
	"github.com/fizzadar/cache.fyi/internal/util"
	"github.com/rs/zerolog"
)

var _ ContentProcessor = (*LinkwardenContentProcessor)(nil)

type LinkwardenContentProcessor struct {
	config config.CachefyiConfig
	log    zerolog.Logger
	db     *database.Database
	client *http.Client
}

func NewLinkwardenContentProcessor(
	cfg config.CachefyiConfig,
	log zerolog.Logger,
	db *database.Database,
) *LinkwardenContentProcessor {
	log = log.With().Str("processor", "linkwarden").Logger()

	return &LinkwardenContentProcessor{
		config: cfg,
		log:    log,
		db:     db,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (lcp *LinkwardenContentProcessor) ProcessContent(ctx context.Context, content *types.Content) error {
	// Only process URL content that doesn't already have a Linkwarden link ID
	if content.Type != types.ContentTypeURL || content.URL == "" {
		return nil
	}

	if content.LinkwardenLinkID != nil {
		lcp.log.Debug().Int64("content_id", content.ID).Msg("Content already has Linkwarden link ID, skipping")
		return nil
	}

	// Check if Linkwarden is configured
	if lcp.config.LinkwardenURL == "" || lcp.config.LinkwardenAPIToken == "" {
		lcp.log.Debug().Msg("Linkwarden not configured, skipping")
		return nil
	}

	lcp.log.Info().Int64("content_id", content.ID).Str("url", content.URL).Msg("Creating Linkwarden link")

	// Create the link in Linkwarden
	linkID, err := lcp.createLinkwardenLink(ctx, content)
	if err != nil {
		return fmt.Errorf("failed to create Linkwarden link: %w", err)
	}

	// Update the content with the Linkwarden link ID
	if err := lcp.db.SetContentLinkwardenID(ctx, content.ID, fmt.Sprintf("%d", linkID)); err != nil {
		return fmt.Errorf("failed to set Linkwarden link ID: %w", err)
	}

	lcp.log.Info().
		Int64("content_id", content.ID).
		Int64("linkwarden_id", linkID).
		Msg("Successfully created Linkwarden link")
	return nil
}

type linkwardenCreateLinkRequest struct {
	URL  string `json:"url"`
	Name string `json:"name,omitempty"`
}

type linkwardenCreateLinkResponse struct {
	Response struct {
		ID int64 `json:"id"`
	} `json:"response"`
}

func (lcp *LinkwardenContentProcessor) createLinkwardenLink(ctx context.Context, content *types.Content) (int64, error) {
	reqBody := linkwardenCreateLinkRequest{
		URL:  content.URL,
		Name: content.URL, // Use URL as name if no title is available
	}

	req := util.LinkwardenRequest{
		APIToken: lcp.config.LinkwardenAPIToken,
		URL:      lcp.config.LinkwardenURL,
		Endpoint: "/api/v1/links",
		Method:   http.MethodPost,
		Body:     reqBody,
	}

	resp, err := util.DoLinkwardenRequest(ctx, req)
	if err != nil {
		return 0, err
	}

	var linkResp linkwardenCreateLinkResponse
	if err := json.Unmarshal(resp.Body, &linkResp); err != nil {
		return 0, err
	}

	return linkResp.Response.ID, nil
}
