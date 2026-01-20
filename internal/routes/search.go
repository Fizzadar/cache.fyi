package routes

import (
	"html/template"
	"net/http"
	"strconv"

	"github.com/fizzadar/cache.fyi/internal/database"
)

func (rt *Routes) initSearchTemplates() error {
	var err error

	rt.searchTemplate, err = template.ParseFS(templates, "templates/search.html", "templates/base.html")
	if err != nil {
		return err
	}

	return nil
}

func (rt *Routes) Search(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	limitStr := r.URL.Query().Get("l")
	limit, _ := strconv.Atoi(limitStr)
	if limit == 0 {
		limit = 50
	}

	var results *database.SearchResults
	var err error

	if query != "" {
		results, err = rt.db.Search(r.Context(), query, limit)
		if err != nil {
			rt.unknownErrorResponse(w, r, "Error performing search", err)
			return
		}
	}

	templateResponse(w, http.StatusOK, rt.searchTemplate, struct {
		Section string
		Query   string
		Limit   int
		Results *database.SearchResults
	}{"search", query, limit, results})
}
