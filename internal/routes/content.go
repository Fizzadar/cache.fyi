package routes

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/fizzadar/cache.fyi/internal/types"
	"github.com/rs/zerolog/hlog"
)

func (rt *Routes) initContentTemplates() error {
	var err error

	rt.listContentPageTemplate, err = template.ParseFS(templates, "templates/content/list.html", "templates/base.html")
	if err != nil {
		return err
	}

	rt.listContentAutoTagsTemplate, err = template.ParseFS(templates, "templates/content/autotags.html", "templates/base.html")
	if err != nil {
		return err
	}

	return nil
}

func (rt *Routes) ListContent(w http.ResponseWriter, r *http.Request) {
	contents, err := rt.db.ListContent(r.Context())
	if err != nil {
		rt.unknownErrorResponse(w, r, "Error fetching content", err)
		return
	}

	tags, err := rt.db.ListTags(r.Context())
	if err != nil {
		rt.unknownErrorResponse(w, r, "Error getting tags", err)
		return
	}

	templateResponse(w, http.StatusOK, rt.listContentPageTemplate, struct {
		Section    string
		Subsection string
		Contents   []*types.Content
		Tags       []types.Tag
	}{"content", "", contents, tags})
}

func (rt *Routes) ListContentURLs(w http.ResponseWriter, r *http.Request) {
	contents, err := rt.db.ListContent(r.Context())
	if err != nil {
		rt.unknownErrorResponse(w, r, "Error fetching content", err)
		return
	}

	urls := make([]string, 0)

	for _, content := range contents {
		if content.URL != "" {
			urls = append(urls, content.URL)
		}
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(strings.Join(urls, "\n")))
}

func (rt *Routes) CreateContent(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(1024 * 1024 * 10) // 10MB max in memory

	cType := types.ContentType(r.PostForm.Get("type"))
	rawType := r.PostForm.Get("type_raw")

	log := hlog.FromRequest(r).With().
		Str("type", string(cType)).
		Str("type_raw", rawType).
		Logger()
	log.Info().Msg("Creating content")

	autoTags, err := rt.db.ListContentAutoTags()
	if err != nil {
		rt.unknownErrorResponse(w, r, "Error fetching auto tags", err)
		return
	}

	urlStr := r.PostForm.Get("url")
	tagIDs := make([]int64, 0)

	if urlStr != "" {
		// Safari shares include a bunch of additional test from the webpage (no idea why), so ensure
		// we strip any of that away before checking the URL is valid.
		urlStr = strings.Split(strings.ReplaceAll(urlStr, "\r\n", "\n"), "\n")[0]
		if _, err := url.Parse(urlStr); err != nil {
			rt.errorResponse(w, r, "URL is invalid", http.StatusBadRequest)
			return
		}

		// Find matching tag IDs
		for _, autoTag := range autoTags {
			if matched, err := regexp.MatchString(autoTag.URLRegex, urlStr); err != nil {
				rt.unknownErrorResponse(w, r, "Error parsing regex", err)
				return
			} else if matched {
				tagIDs = append(tagIDs, autoTag.TagID)
			}
		}
		// Remove duplicates
		slices.Sort(tagIDs)
		tagIDs = slices.Compact(tagIDs)
	}

	switch cType {
	case types.ContentTypeURL:
		if urlStr == "" {
			rt.errorResponse(w, r, "URL is required", http.StatusBadRequest)
			return
		}
		if id, err := rt.db.CreateURLContent(r.Context(), urlStr, tagIDs, nil); err != nil {
			rt.unknownErrorResponse(w, r, "Error creating content", err)
			return
		} else {
			log.Info().Int64("id", id).Msg("Created content")
			if r.PostForm.Has("redirect") {
				http.Redirect(w, r, "/content", http.StatusSeeOther)
				return
			} else {
				w.Write([]byte(strconv.Itoa(int(id))))
				return
			}
		}

	case types.ContentTypeFile:
		file, header, err := r.FormFile("file")
		if err != nil {
			rt.unknownErrorResponse(w, r, "Error parsing file", err)
			return
		} else if file == nil {
			rt.errorResponse(w, r, "No file included in request", http.StatusBadRequest)
			return
		} else if header == nil {
			rt.errorResponse(w, r, "No file header included in request", http.StatusBadRequest)
			return
		}

		defer file.Close()
		data, err := io.ReadAll(file)
		if err != nil {
			rt.unknownErrorResponse(w, r, "Error reading file", err)
			return
		}

		hashB := sha256.Sum256(data)
		hash := base64.RawURLEncoding.EncodeToString(hashB[:])
		contentType := header.Header.Get("Content-Type")

		if id, err := rt.db.CreateContent(r.Context(), types.ContentTypeFile, hash, urlStr, data, &contentType, &header.Filename, tagIDs, nil); err != nil {
			rt.unknownErrorResponse(w, r, "Error creating content", err)
			return
		} else {
			log.Info().Int64("id", id).Msg("Created content")
			if r.PostForm.Has("redirect") {
				http.Redirect(w, r, "/content", http.StatusSeeOther)
				return
			} else {
				w.Write([]byte(strconv.Itoa(int(id))))
				return
			}
		}

	default:
		rt.errorResponse(w, r, fmt.Sprintf("Invalid content type: %s", cType), http.StatusBadRequest)
		return
	}
}

func (rt *Routes) GetContentFile(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, _ := strconv.Atoi(idStr)

	content, err := rt.db.GetContent(r.Context(), int64(id))
	if err != nil {
		rt.unknownErrorResponse(w, r, "Error getting content", err)
		return
	} else if content == nil {
		rt.errorResponse(w, r, "No content found", http.StatusNotFound)
		return
	}

	sizeBytes := content.SizeBytes

	contentType := "application/octet-stream"
	if content.ContentType != nil {
		contentType = *content.ContentType
	}

	filename := "unknown"
	if content.Filename != nil {
		filename = *content.Filename
	}

	data, err := rt.db.GetContentData(r.Context(), int64(id))
	if err != nil {
		rt.unknownErrorResponse(w, r, "Error getting content data", err)
		return
	} else if data == nil {
		data = []byte(content.URL)
		sizeBytes = len(data)
		contentType = "text/plain"
	}

	w.Header().Add("Content-Type", contentType)
	w.Header().Add("Content-Length", strconv.Itoa(sizeBytes))
	w.Header().Add("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, filename))
	w.WriteHeader(http.StatusOK)

	w.Write(data)
}

func (rt *Routes) AddContentTags(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		rt.errorResponse(w, r, "Error pasing form", http.StatusBadRequest)
		return
	}

	contentIDs := []int64{}
	for _, cidStr := range r.PostForm["content-id"] {
		id, err := strconv.Atoi(cidStr)
		if err != nil {
			rt.errorResponse(w, r, "Invalid content ID", http.StatusBadRequest)
			return
		}
		contentIDs = append(contentIDs, int64(id))
	}

	tag := r.PostForm.Get("tag")
	var tagID int64
	var err error
	if tag == "NEW" {
		tagName := r.PostForm.Get("tagName")
		tagID, err = rt.db.EnsureTag(r.Context(), tagName)
		if err != nil {
			rt.unknownErrorResponse(w, r, "Error getting tag ID", err)
			return
		}
	} else {
		if tid, err := strconv.Atoi(tag); err != nil {
			rt.errorResponse(w, r, "Invalid tag ID", http.StatusBadRequest)
			return
		} else {
			tagID = int64(tid)
		}
	}

	for _, cid := range contentIDs {
		if err := rt.db.AddContentTag(r.Context(), cid, tagID); err != nil {
			rt.unknownErrorResponse(w, r, "Error setting content tag", err)
		}
	}

	http.Redirect(w, r, "/content", http.StatusSeeOther)
}
