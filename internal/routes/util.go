package routes

import (
	"html/template"
	"net/http"

	"github.com/rs/zerolog/hlog"
)

func templateResponse(w http.ResponseWriter, statusCode int, template *template.Template, data any) {
	w.WriteHeader(statusCode)
	w.Header().Set("Content-Type", "text/html")
	if err := template.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (rt *Routes) errorResponse(w http.ResponseWriter, r *http.Request, message string, statusCode int) {
	hlog.FromRequest(r).Warn().Int("status_code", statusCode).Msg(message)
	templateResponse(w, statusCode, rt.errorTemplate, struct {
		Section     string
		StatusCode  int
		Message     string
		RequestPath string
	}{"", statusCode, message, r.URL.Path})
}

func (rt *Routes) unknownErrorResponse(w http.ResponseWriter, r *http.Request, message string, err error) {
	hlog.FromRequest(r).Err(err).Int("status_code", http.StatusInternalServerError).Msg(message)
	templateResponse(w, http.StatusInternalServerError, rt.errorTemplate, struct {
		Section     string
		StatusCode  int
		Message     string
		Error       error
		RequestPath string
	}{"", http.StatusInternalServerError, message, err, r.URL.Path})
}
