package routes

import (
	"html/template"
	"net/http"
)

func (rt *Routes) initStatsTemplates() error {
	var err error
	rt.getStatsTemplate, err = template.ParseFS(templates, "templates/stats.html", "templates/base.html")
	return err
}

func (rt *Routes) GetStats(w http.ResponseWriter, r *http.Request) {
	stats, err := rt.db.GetStats(r.Context())
	if err != nil {
		rt.unknownErrorResponse(w, r, "Error fetching database stats", err)
		return
	}

	templateResponse(w, http.StatusOK, rt.getStatsTemplate, struct {
		Section string
		Stats   interface{}
	}{"stats", stats})
}
