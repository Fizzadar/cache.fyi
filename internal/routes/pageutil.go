package routes

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"net/http"
	"regexp"
	"strconv"
	"strings"

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

var tagContent = regexp.MustCompile(`\[\[[a-zA-Z0-9#\/\-]+\]\]`)

func (rt *Routes) handleContentTag(ctx context.Context, s string) string {
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
	case types.ContentTypeURL:
		return "[content#" + idStr + "](" + url + "): " + content.URL
	case types.ContentTypeFile:
		url = "/c/f/" + idStr
		if content.Filename != nil {
			name = *content.Filename
		} else {
			name = "file with no name"
		}
		var contentType string
		if content.ContentType != nil {
			contentType = *content.ContentType
		}
		name += " (" + contentType + ", " + strconv.Itoa(content.SizeBytes) + " bytes)"
		link := "[content#" + idStr + "](" + url + "): " + name

		switch contentType {
		case "image/jpeg", "image/png", "image/gif", "image/webp", "image/apng":
			link = link + "\n\n![](" + url + ")"
		case "application/json":
			data, err := rt.db.GetContentData(ctx, int64(id))
			if err != nil {
				link = link + "\n\nerror getting data: " + err.Error()
			} else {
				link = link + "\n\n```\n" + string(data) + "\n```\n"
			}
		}
		return link
	default:
		return fmt.Sprintf("unknown content type: %s", content.Type)
	}
}

func (rt *Routes) handlePageTag(ctx context.Context, s string) string {
	path := s[2 : len(s)-2]

	page, err := rt.db.GetPage(ctx, path)
	if err != nil {
		return fmt.Errorf("error getting page with path: %s: %w", path, err).Error()
	} else if page == nil {
		return fmt.Errorf("no page found at path: %s", path).Error()
	}

	title := page.Title
	if title == "" {
		title = path
	}

	return "[" + title + "](/p" + path + ")"
}

func (rt *Routes) handleTagTag(ctx context.Context, s string) string {
	tag := s[6 : len(s)-2]
	return "[#" + tag + "](/t/" + tag + ")"
}

func (rt *Routes) getPageContent(ctx context.Context, p *types.Page) (template.HTML, error) {
	content := tagContent.ReplaceAllStringFunc(p.Content, func(s string) string {
		if strings.HasPrefix(s, "[[content#") {
			s = rt.handleContentTag(ctx, s)
		} else if strings.HasPrefix(s, "[[tag#") {
			s = rt.handleTagTag(ctx, s)
		} else {
			s = rt.handlePageTag(ctx, s)
		}
		return s
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
