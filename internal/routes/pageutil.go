package routes

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"net/http"
	"regexp"
	"strconv"

	"github.com/fizzadar/cache.fyi/internal/types"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

func getPagePathFromRequest(r *http.Request) string {
	// Extract the path and name from the pathname parameter
	path := r.PathValue("pathname")
	// Prepend the "/" to path since the router removes it
	path = "/" + path
	return path
}

var contentRegex = regexp.MustCompile(`\[\[content#[0-9]+\]\]`)

func (rt *Routes) getPageContent(ctx context.Context, p *types.Page) (template.HTML, error) {
	content := contentRegex.ReplaceAllStringFunc(p.Content, func(s string) string {
		idStr := s[10 : len(s)-2]
		id, err := strconv.Atoi(idStr)
		if err != nil {
			return fmt.Errorf("error parsing content ID: %w", err).Error()
		}
		content, err := rt.db.GetContent(ctx, int64(id))
		if err != nil {
			return fmt.Errorf("error getting content with ID: %d: %w", id, err).Error()
		}

		var url, name string

		switch content.Type {
		case types.ContentTypeFile:
			url = "/c/f/" + idStr
			if content.Filename != nil {
				name = *content.Filename
			} else {
				name = "file with no name"
			}
			name += " (" + strconv.Itoa(content.SizeBytes) + " bytes)"
		case types.ContentTypeURL:
			name = content.URL
			url = content.URL
		default:
			return fmt.Sprintf("unknown content type: %s", content.Type)
		}

		return "[content#" + idStr + ":" + name + "](" + url + ")"
	})

	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			html.WithUnsafe(),
		),
	)

	var buf bytes.Buffer
	if err := md.Convert([]byte(content), &buf); err != nil {
		return "", err
	}
	return template.HTML(buf.Bytes()), nil
}
