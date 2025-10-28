package routes

import (
	"embed"
	"html/template"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/fizzadar/cache.fyi/internal/config"
	"github.com/fizzadar/cache.fyi/internal/database"
	"github.com/fizzadar/cache.fyi/internal/types"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
)

//go:embed templates
var templates embed.FS

//go:embed static
var static embed.FS

const authHeaderKey = "Authorization"

type Routes struct {
	config config.CachefyiConfig
	log    zerolog.Logger
	db     *database.Database
	server *http.Server

	errorTemplate,
	getHomeTemplate,
	getPageTemplate,
	listAllPagesPageTemplate,
	listTagsTemplate,
	getTagTemplate,
	listContentPageTemplate,
	listContentAutoTagsTemplate,
	listPageAutoTagsTemplate,
	getStatsTemplate *template.Template
}

func NewRoutes(cfg config.CachefyiConfig, log zerolog.Logger, db *database.Database) *Routes {
	log = log.With().Str("component", "routes").Logger()

	routes := &Routes{
		config: cfg,
		log:    log,
		db:     db,
	}

	if err := routes.initPageTemplates(); err != nil {
		panic(err)
	} else if err = routes.initContentTemplates(); err != nil {
		panic(err)
	} else if err := routes.initTagTemplates(); err != nil {
		panic(err)
	} else if err := routes.initStatsTemplates(); err != nil {
		panic(err)
	}

	var err error
	routes.errorTemplate, err = template.ParseFS(templates, "templates/error.html", "templates/base.html")
	if err != nil {
		panic(err)
	}
	routes.getHomeTemplate, err = template.ParseFS(templates, "templates/home.html", "templates/base.html")
	if err != nil {
		panic(err)
	}

	// Add recovery middleware
	recoveryHandler := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					log.Error().Msgf("Panic recovered: %v\nStack trace:\n%s", err, debug.Stack())
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}

	// Add authheader middleware
	authHandler := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get(authHeaderKey)
			if authHeader == "" {
				if cookie, err := r.Cookie(authHeaderKey); err == nil {
					authHeader = cookie.Value
				}
			}
			if authHeader != cfg.AuthHeader {
				routes.errorResponse(w, r, "Invalid auth header", http.StatusForbidden)
				return
			}
			next(w, r)
		}
	}

	mux := http.NewServeMux()

	// Public routes
	mux.Handle("GET /static/", http.FileServerFS(static))
	mux.HandleFunc("POST /login", routes.Login)

	// Private routes
	mux.HandleFunc("GET /{$}", authHandler(routes.GetHome))
	mux.HandleFunc("GET /logout", authHandler(routes.Logout))

	// Content
	mux.HandleFunc("GET /content", authHandler(routes.ListContent))
	mux.HandleFunc("GET /content/archivebox", authHandler(routes.ListContentURLs))
	mux.HandleFunc("POST /content", authHandler(routes.CreateContent))
	mux.HandleFunc("GET /content/autotags", authHandler(routes.ListContentAutoTags))
	mux.HandleFunc("POST /content/autotags", authHandler(routes.CreateContentAutoTag))
	mux.HandleFunc("POST /content/autotags/delete", authHandler(routes.DeleteContentAutoTag))
	mux.HandleFunc("POST /content/tags", authHandler(routes.AddContentTags))
	// mux.HandleFunc("POST /content/autotags/eval", authHandler(routes.EvalContentAutoTag))
	mux.HandleFunc("GET /c/f/{id}", authHandler(routes.GetContentFile))

	// Pages
	mux.HandleFunc("GET /p/{pathname...}", authHandler(routes.GetPage))
	mux.HandleFunc("POST /p/{pathname...}", authHandler(routes.CreateOrUpsertPage))
	mux.HandleFunc("GET /pages", authHandler(routes.ListAllPages))
	mux.HandleFunc("GET /pages/{id}", authHandler(routes.RedirectToPageByID))
	mux.HandleFunc("POST /pages/tags", authHandler(routes.AddPageTag))
	mux.HandleFunc("POST /pages/tags/delete", authHandler(routes.DeletePageTag))
	mux.HandleFunc("GET /pages/autotags", authHandler(routes.ListPageAutoTags))
	mux.HandleFunc("POST /pages/autotags", authHandler(routes.CreatePageAutoTag))
	mux.HandleFunc("POST /pages/autotags/delete", authHandler(routes.DeletePageAutoTag))

	// Tags
	mux.HandleFunc("GET /tags", authHandler(routes.ListTags))
	mux.HandleFunc("POST /tags", authHandler(routes.CreateTag))
	mux.HandleFunc("POST /tags/delete", authHandler(routes.DeleteTag))
	mux.HandleFunc("GET /t/{tagName}", authHandler(routes.GetTag))

	// Stats
	mux.HandleFunc("GET /stats", authHandler(routes.GetStats))

	// Add zerolog middleware
	handler := hlog.NewHandler(log)(mux)
	handler = hlog.AccessHandler(func(r *http.Request, status, size int, duration time.Duration) {
		log.Info().
			Str("method", r.Method).
			Stringer("url", r.URL).
			Int("status", status).
			Int("size", size).
			Dur("duration", duration).
			Msg("Request")
	})(handler)
	handler = hlog.RemoteAddrHandler("ip")(handler)
	handler = hlog.UserAgentHandler("user_agent")(handler)
	handler = hlog.RefererHandler("referer")(handler)
	handler = hlog.RequestIDHandler("req_id", "Request-Id")(handler)

	// Apply recovery middleware to the router last (so applied first)
	handler = recoveryHandler(handler)

	routes.server = &http.Server{
		Addr:    cfg.ListenAddr, // You may want to make this configurable
		Handler: handler,
	}
	return routes
}

func (r *Routes) Start() {
	go func() {
		r.log.Info().Str("listenAddr", r.config.ListenAddr).Msg("Starting routes...")
		if err := r.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			r.log.Fatal().Msgf("Server error: %v", err)
		}
	}()
}

func (rt *Routes) GetHome(w http.ResponseWriter, r *http.Request) {
	pages, err := rt.db.ListPinnedPages(r.Context())
	if err != nil {
		rt.unknownErrorResponse(w, r, "Error fetching pinned pages", err)
		return
	}

	templateResponse(w, http.StatusOK, rt.getHomeTemplate, struct {
		Section string
		Pages   []*types.Page
	}{"home", pages})
}

// "Login" which just sets the authHeader cookie + redirects
func (rt *Routes) Login(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	http.SetCookie(w, &http.Cookie{
		Name:     authHeaderKey,
		Value:    r.PostForm.Get("authHeader"),
		Path:     "/",
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteStrictMode,
	})

	dest := r.PostForm.Get("path")
	if dest == "" {
		dest = "/"
	}
	http.Redirect(w, r, dest, http.StatusSeeOther)
}

func (rt *Routes) Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:    authHeaderKey,
		Expires: time.Unix(0, 0),
	})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
