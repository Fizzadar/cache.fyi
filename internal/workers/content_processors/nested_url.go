package content_processors

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/fizzadar/cache.fyi/internal/config"
	"github.com/fizzadar/cache.fyi/internal/database"
	"github.com/fizzadar/cache.fyi/internal/types"
	"github.com/rs/zerolog"
)

var _ ContentProcessor = (*NestedURLContentProcessor)(nil)

type NestedURLContentProcessor struct {
	config config.CachefyiConfig
	log    zerolog.Logger
	db     *database.Database
}

func NewNestedURLContentProcessor(
	cfg config.CachefyiConfig,
	log zerolog.Logger,
	db *database.Database,
) *NestedURLContentProcessor {
	log = log.With().Str("processor", "nested_url_processor").Logger()

	return &NestedURLContentProcessor{
		config: cfg,
		log:    log,
		db:     db,
	}
}

func (ncp *NestedURLContentProcessor) ProcessContent(ctx context.Context, content *types.Content) error {
	ncp.log.Trace().Int64("content_id", content.ID).Msg("Process content")

	if content.URL == "" {
		ncp.log.Trace().Msg("Content has no url")
		return nil
	}

	url, err := url.Parse(content.URL)
	if err != nil {
		return fmt.Errorf("failed to parse content URL: %w", err)
	}

	if url.Host == "news.ycombinator.com" {
		if err := ncp.processHackerNewsURL(content, url); err != nil {
			return err
		}
	} else {
		ncp.log.Trace().Any("host", url.Host).Msg("URL host is not hacker news")
	}

	return nil
}

func (ncp *NestedURLContentProcessor) processHackerNewsURL(content *types.Content, url *url.URL) error {
	if url.Path != "/item" {
		ncp.log.Trace().Any("url", url).Msg("Url does not have path=/item")
		return nil
	}

	itemID := url.Query().Get("id")
	if itemID == "" {
		ncp.log.Trace().Any("url", url).Msg("Url does not have id query argument")
		return nil
	}

	resp, err := http.DefaultClient.Get(fmt.Sprintf("https://hacker-news.firebaseio.com/v0/item/%s.json", itemID))
	if err != nil {
		return err
	} else {
		defer resp.Body.Close()
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("got invalid status code from API: %d", resp.StatusCode)
	}

	var data = struct {
		URL string `json:"url"`
	}{}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return err
	} else if data.URL == "" {
		return nil
	}

	ncp.log.Debug().Str("url", data.URL).Msg("Adding nested Hacker News URL")
	if _, err := ncp.db.CreateURLContent(context.TODO(), data.URL, nil, &content.ID); err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed: content.hash") {
			ncp.log.Warn().Str("url", data.URL).Msg("Duplicate nested URL found in processing")
			return nil
		}
		return err
	}

	return nil
}
