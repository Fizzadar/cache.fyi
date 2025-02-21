package routes

import (
	"net/http"
	"strconv"

	"github.com/fizzadar/cache.fyi/internal/types"
)

func (rt *Routes) CreateContentAutoTag(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	tagID, err := strconv.Atoi(r.PostForm.Get("tag_id"))
	if err != nil {
		rt.errorResponse(w, r, "Invalid tag ID", http.StatusBadRequest)
		return
	}

	urlRegex := r.PostForm.Get("url_regex")
	if urlRegex == "" {
		rt.errorResponse(w, r, "URL regex is required", http.StatusBadRequest)
		return
	}

	_, err = rt.db.CreateContentAutoTag(tagID, urlRegex)
	if err != nil {
		rt.unknownErrorResponse(w, r, "Error creating content auto tag", err)
		return
	}

	http.Redirect(w, r, "/content/autotags", http.StatusSeeOther)
}

func (rt *Routes) ListContentAutoTags(w http.ResponseWriter, r *http.Request) {
	tags, err := rt.db.ListTags(r.Context())
	if err != nil {
		rt.unknownErrorResponse(w, r, "Error fetching tags", err)
		return
	}

	autoTags, err := rt.db.ListContentAutoTags()
	if err != nil {
		rt.unknownErrorResponse(w, r, "Error fetching content auto tags", err)
		return
	}

	templateResponse(w, http.StatusOK, rt.listContentAutoTagsTemplate, struct {
		Section    string
		Subsection string
		Tags       []types.Tag
		AutoTags   []types.ContentAutoTag
	}{"content", "autotags", tags, autoTags})
}

func (rt *Routes) DeleteContentAutoTag(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	id, err := strconv.Atoi(r.PostForm.Get("tag_id"))
	if err != nil {
		rt.errorResponse(w, r, "Invalid tag ID", http.StatusBadRequest)
		return
	}

	err = rt.db.DeleteContentAutoTag(id)
	if err != nil {
		rt.unknownErrorResponse(w, r, "Error deleting content auto tag", err)
		return
	}

	http.Redirect(w, r, "/content/autotags", http.StatusSeeOther)
}

func (rt *Routes) EvalContentAutoTag(w http.ResponseWriter, r *http.Request) {

}
