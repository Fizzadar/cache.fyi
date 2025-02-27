package routes

import (
	"net/http"
	"strconv"

	"html/template"

	"github.com/fizzadar/cache.fyi/internal/types"
	"github.com/rs/zerolog/hlog"
)

func (rt *Routes) initPageTemplates() error {
	var err error

	rt.getPageTemplate, err = template.ParseFS(templates, "templates/page/get.html", "templates/base.html")
	if err != nil {
		return err
	}

	rt.listAllPagesPageTemplate, err = template.ParseFS(templates, "templates/page/list.html", "templates/base.html")
	if err != nil {
		return err
	}

	rt.listPageAutoTagsTemplate, err = template.ParseFS(templates, "templates/page/autotags.html", "templates/base.html")
	if err != nil {
		return err
	}

	return nil
}

func (rt *Routes) ListAllPages(w http.ResponseWriter, r *http.Request) {
	pages, err := rt.db.ListPages(r.Context())
	if err != nil {
		rt.unknownErrorResponse(w, r, "Error fetching pages", err)
		return
	}

	templateResponse(w, http.StatusOK, rt.listAllPagesPageTemplate, struct {
		Section    string
		Subsection string
		Pages      []*types.Page
	}{"pages", "", pages})
}

func (rt *Routes) RedirectToPageByID(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		rt.errorResponse(w, r, "Invalid page ID", http.StatusBadRequest)
		return
	}

	path, err := rt.db.GetPagePathFromID(r.Context(), int64(id))
	if err != nil {
		rt.unknownErrorResponse(w, r, "Error fetching page", err)
		return
	}

	http.Redirect(w, r, "/p"+path, http.StatusSeeOther)
}

func (rt *Routes) GetPage(w http.ResponseWriter, r *http.Request) {
	path := getPagePathFromRequest(r)

	if path == "/" {
		http.Redirect(w, r, "/pages", http.StatusSeeOther)
		return
	}

	page, err := rt.db.GetPage(r.Context(), path)
	if err != nil {
		rt.unknownErrorResponse(w, r, "Error fetching page", err)
		return
	} else if page == nil {
		hlog.FromRequest(r).Warn().Str("path", path).Msg("Page not found")
	}

	var html template.HTML
	if page != nil {
		html, err = rt.getPageContent(r.Context(), page)
		if err != nil {
			rt.unknownErrorResponse(w, r, "Error getting page html", err)
			return
		}
	}

	tags, err := rt.db.ListTags(r.Context())
	if err != nil {
		rt.unknownErrorResponse(w, r, "Error getting tags", err)
		return
	}

	otherPagesByTag := make(map[string][]*types.Page, 10)
	if page != nil {
		for _, tagName := range page.Tags {
			tagID, err := rt.db.TagIDForName(r.Context(), tagName)
			if err != nil {
				rt.unknownErrorResponse(w, r, "Error fetching tag", err)
				return
			}
			pages, err := rt.db.ListPagesForTagID(r.Context(), tagID)
			if err != nil {
				rt.unknownErrorResponse(w, r, "Error fetching pages", err)
				return
			}
			otherPages := make([]*types.Page, 0, len(pages))
			for _, p := range pages {
				if p.Path != page.Path {
					otherPages = append(otherPages, p)
				}
			}
			if len(otherPages) > 0 {
				otherPagesByTag[tagName] = otherPages
			}
		}
	}

	templateResponse(w, http.StatusOK, rt.getPageTemplate, struct {
		Section         string
		Path            string
		Page            *types.Page
		HTMLContent     template.HTML
		Tags            []types.Tag
		OtherPagesByTag map[string][]*types.Page
	}{"page", path, page, html, tags, otherPagesByTag})
}

func (rt *Routes) CreateOrUpsertPage(w http.ResponseWriter, r *http.Request) {
	path := getPagePathFromRequest(r)

	// Parse the form data
	if err := r.ParseForm(); err != nil {
		rt.errorResponse(w, r, "Error parsing form", http.StatusBadRequest)
		return
	}

	title := r.PostForm.Get("title")
	content := r.PostForm.Get("content")
	if content == "" {
		rt.errorResponse(w, r, "Content is required", http.StatusBadRequest)
		return
	}

	if err := rt.db.UpsertPage(r.Context(), path, title, content); err != nil {
		rt.unknownErrorResponse(w, r, "Error updating page", err)
		return
	}

	// Redirect back to the GetPage route
	http.Redirect(w, r, r.URL.Path, http.StatusSeeOther)
}

func (rt *Routes) AddPageTag(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		rt.errorResponse(w, r, "Error pasing form", http.StatusBadRequest)
		return
	}

	pageIDStr := r.PostForm.Get("pageID")
	pageID, err := strconv.Atoi(pageIDStr)
	if err != nil {
		rt.errorResponse(w, r, "Invalid page ID", http.StatusBadRequest)
		return
	}

	tag := r.PostForm.Get("tag")
	var tagID int64
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

	if err := rt.db.AddPageTag(r.Context(), int64(pageID), tagID); err != nil {
		rt.unknownErrorResponse(w, r, "Error adding page tag", err)
		return
	}

	// Redirect back to the GetPage route
	http.Redirect(w, r, "/pages/"+pageIDStr, http.StatusSeeOther)
}

func (rt *Routes) DeletePageTag(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		rt.errorResponse(w, r, "Error pasing form", http.StatusBadRequest)
		return
	}

	pageIDStr := r.PostForm.Get("pageID")
	pageID, err := strconv.Atoi(pageIDStr)
	if err != nil {
		rt.errorResponse(w, r, "Invalid page ID", http.StatusBadRequest)
		return
	}

	tagName := r.PostForm.Get("tag")
	tagID, err := rt.db.TagIDForName(r.Context(), tagName)
	if err != nil {
		rt.unknownErrorResponse(w, r, "Error getting tag ID", err)
		return
	}

	if err := rt.db.DeletePageTag(r.Context(), int64(pageID), tagID); err != nil {
		rt.unknownErrorResponse(w, r, "Error deleting page tag", err)
		return
	}

	// Redirect back to the GetPage route
	http.Redirect(w, r, "/pages/"+pageIDStr, http.StatusSeeOther)
}
