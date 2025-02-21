package routes

import (
	"html/template"
	"net/http"
	"strconv"

	"github.com/fizzadar/cache.fyi/internal/types"
)

func (rt *Routes) initTagTemplates() error {
	var err error

	rt.getTagTemplate, err = template.ParseFS(templates, "templates/tag/get.html", "templates/base.html")
	if err != nil {
		return err
	}

	rt.listTagsTemplate, err = template.ParseFS(templates, "templates/tag/list.html", "templates/base.html")
	if err != nil {
		return err
	}

	return nil
}

func (rt *Routes) ListTags(w http.ResponseWriter, r *http.Request) {
	tags, err := rt.db.ListTags(r.Context())
	if err != nil {
		rt.unknownErrorResponse(w, r, "Error fetching tags", err)
		return
	}

	templateResponse(w, http.StatusOK, rt.listTagsTemplate, struct {
		Section string
		Tag     string
		Tags    []types.Tag
	}{"tags", "", tags})
}

func (rt *Routes) GetTag(w http.ResponseWriter, r *http.Request) {
	tagName := r.PathValue("tagName")

	tagID, err := rt.db.TagIDForName(r.Context(), tagName)
	if err != nil {
		rt.unknownErrorResponse(w, r, "Error fetching tag", err)
		return
	}

	content, err := rt.db.ListContentForTagID(r.Context(), tagID)
	if err != nil {
		rt.unknownErrorResponse(w, r, "Error fetching content", err)
		return
	}

	pages, err := rt.db.ListPagesForTagID(r.Context(), tagID)
	if err != nil {
		rt.unknownErrorResponse(w, r, "Error fetching pages", err)
		return
	}

	templateResponse(w, http.StatusOK, rt.getTagTemplate, struct {
		Section  string
		Tag      string
		Contents []*types.Content
		Pages    []*types.Page
	}{"tags", tagName, content, pages})
}

func (rt *Routes) CreateTag(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	name := r.PostForm.Get("name")
	if name == "" {
		rt.errorResponse(w, r, "Name must be set", http.StatusBadRequest)
		return
	}

	if _, err := rt.db.CreateTag(name); err != nil {
		rt.unknownErrorResponse(w, r, "Error creating tag", err)
		return
	}

	http.Redirect(w, r, "/tags", http.StatusSeeOther)
}

func (rt *Routes) DeleteTag(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	id, err := strconv.Atoi(r.PostForm.Get("id"))
	if err != nil {
		rt.errorResponse(w, r, "Name invalid", http.StatusBadRequest)
		return
	} else if id == 0 {
		rt.errorResponse(w, r, "Name must be set", http.StatusBadRequest)
		return
	}

	if err := rt.db.DeleteTag(id); err != nil {
		rt.unknownErrorResponse(w, r, "Error deleting tag", err)
		return
	}

	http.Redirect(w, r, "/tags", http.StatusSeeOther)
}
