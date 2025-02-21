package routes

import (
	"net/http"
	"strconv"

	"github.com/fizzadar/cache.fyi/internal/types"
)

func (rt *Routes) CreatePageAutoTag(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	tagID, err := strconv.Atoi(r.PostForm.Get("tag_id"))
	if err != nil {
		rt.errorResponse(w, r, "Invalid tag ID", http.StatusBadRequest)
		return
	}

	pathRegex := r.PostForm.Get("path_regex")
	if pathRegex == "" {
		rt.errorResponse(w, r, "Path regex is required", http.StatusBadRequest)
		return
	}

	_, err = rt.db.CreatePageAutoTag(tagID, pathRegex)
	if err != nil {
		rt.unknownErrorResponse(w, r, "Error creating page auto tag", err)
		return
	}

	http.Redirect(w, r, "/pages/autotags", http.StatusSeeOther)
}

func (rt *Routes) ListPageAutoTags(w http.ResponseWriter, r *http.Request) {
	tags, err := rt.db.ListTags(r.Context())
	if err != nil {
		rt.unknownErrorResponse(w, r, "Error fetching tags", err)
		return
	}

	autoTags, err := rt.db.ListPageAutoTags()
	if err != nil {
		rt.unknownErrorResponse(w, r, "Error fetching page auto tags", err)
		return
	}

	templateResponse(w, http.StatusOK, rt.listPageAutoTagsTemplate, struct {
		Section    string
		Subsection string
		Tags       []types.Tag
		AutoTags   []types.PageAutoTag
	}{"pages", "autotags", tags, autoTags})
}

func (rt *Routes) DeletePageAutoTag(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	id, err := strconv.Atoi(r.PostForm.Get("tag_id"))
	if err != nil {
		rt.errorResponse(w, r, "Invalid tag ID", http.StatusBadRequest)
		return
	}

	err = rt.db.DeletePageAutoTag(id)
	if err != nil {
		rt.unknownErrorResponse(w, r, "Error deleting page auto tag", err)
		return
	}

	http.Redirect(w, r, "/pages/autotags", http.StatusSeeOther)
}
