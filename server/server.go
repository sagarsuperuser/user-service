package server

import (
	"context"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/rs/cors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
	"github.com/rs/zerolog/log"

	"github.com/gorilla/mux"
	"github.com/sagarsuperuser/userprofile/internal/httputil"
	"github.com/sagarsuperuser/userprofile/internal/router"
	"github.com/sagarsuperuser/userprofile/internal/versions"
	"github.com/sagarsuperuser/userprofile/server/httpstatus"
	"github.com/sagarsuperuser/userprofile/server/middlewares"
	apiv1 "github.com/sagarsuperuser/userprofile/server/routes/api/v1"
	"github.com/sagarsuperuser/userprofile/server/settings"
	"github.com/sagarsuperuser/userprofile/store"
)

// versionMatcher defines a variable matcher to be parsed by the router
// when a request is about to be served.
const versionMatcher = "/v{version:[0-9.]+}"

type Server struct {
	router      *mux.Router
	settings    *settings.Settings
	store       *store.Store
	httpServer  *http.Server
	httpAddress string
	mu          sync.Mutex
	middlewares []middlewares.Middleware
}

func NewServer(settings *settings.Settings, store *store.Store) *Server {
	ret := new(Server)
	ret.settings = settings
	ret.store = store
	mRouter := mux.NewRouter()

	// Global Middlewares --

	// register panic recovery middleware
	mRouter.Use(middlewares.Recovery())

	// Inject zerolog logger into request context
	mRouter.Use(hlog.NewHandler(log.Logger))

	// Install some provided extra handler to set some request's context fields.
	// Thanks to that handler, all our logs will come with some prepopulated fields.
	mRouter.Use(hlog.RemoteAddrHandler("ip"))
	mRouter.Use(hlog.UserAgentHandler("user_agent"))
	mRouter.Use(hlog.RefererHandler("referer"))
	mRouter.Use(hlog.RequestIDHandler("req_id", ""))

	// register CORS middleware
	c := cors.New(cors.Options{
		AllowedOrigins: settings.Origins,
	})
	mRouter.Use(c.Handler)

	// register access logger, called after each request
	mRouter.Use(hlog.AccessHandler(func(r *http.Request, status, size int, duration time.Duration) {
		hlog.FromRequest(r).Info().
			Str("method", r.Method).
			Stringer("url", r.URL).
			Int("status", status).
			Int("size", size).
			Dur("duration", duration).
			Msg("")
	}))
	// setup listen addresses
	ret.httpAddress = net.JoinHostPort(settings.Host, strconv.Itoa(settings.Port))

	mRouter.Methods(http.MethodGet).Path("/health").HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			hlog.FromRequest(r).Info().Msg("health ok")
			w.WriteHeader(http.StatusOK)
		})

	// register api version 1 endpoints
	apiV1Service := apiv1.NewAPIV1Service(settings, store)
	// TODO- read version from envs/config.
	versionMW, err := middlewares.NewVersionMiddleware("dev", "1.0", "1.0")
	if err != nil {
		log.Fatal().Err(err).Msg("invalid API version configuration")
	}
	ret.UseMiddleware(versionMW)
	ret.router = ret.CreateMux(
		context.Background(),
		mRouter,
		apiv1.NewAuthRouter(apiV1Service),
		apiv1.NewUserRouter(apiV1Service),
	)
	return ret
}

// UseMiddleware registers a global APIFunc middleware.
// They are executed in the order that they are applied to the Router.
func (s *Server) UseMiddleware(mw middlewares.Middleware) {
	s.middlewares = append(s.middlewares, mw)
}

func (s *Server) makeHTTPHandler(r router.Route) http.HandlerFunc {
	handler := r.Handler()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		//  Build APIFunc middleware chain.
		handlerFunc := s.handlerWithGlobalMiddlewares(handler)
		vars := mux.Vars(r)
		if vars == nil {
			vars = make(map[string]string)
		}

		if err := handlerFunc(ctx, w, r, vars); err != nil {
			statusCode := httpstatus.FromError(err)
			respMsg := err.Error()
			if statusCode >= http.StatusInternalServerError {
				// In case of InternalServerError, message sent to client are standard HTTP code messages.
				respMsg = http.StatusText(statusCode)
				zerolog.Ctx(ctx).Error().Err(err).Msgf("Handler for %s %s returned error", r.Method, r.URL.Path)
			}
			// For very old clients expecting plaintext, keep responses readable.
			if v := vars["version"]; v != "" && versions.LessThan(v, "0.1") {
				http.Error(w, respMsg, statusCode)
			} else {
				_ = httputil.WriteRawJSON(w, statusCode, map[string]string{
					"message": respMsg,
				})
			}
		}
	})
}

// CreateMux returns a new mux with all the routers registered.
func (s *Server) CreateMux(ctx context.Context, m *mux.Router, routers ...router.Router) *mux.Router {
	log.Debug().Msg("Registering routers")
	for _, apiRouter := range routers {
		for _, r := range apiRouter.Routes() {
			if ctx.Err() != nil {
				return m
			}
			log.Debug().Str("method", r.Method()).Str("path", r.Path()).Msg("Registering route")
			f := s.makeHTTPHandler(r)
			m.Path(versionMatcher + r.Path()).Methods(r.Method()).Handler(f)
			m.Path(r.Path()).Methods(r.Method()).Handler(f)
		}
	}

	// Setup handlers for undefined paths and methods
	notFoundHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = httputil.WriteRawJSON(w, http.StatusNotFound, map[string]string{
			"message": "page not found",
		})
	})

	m.HandleFunc(versionMatcher+"/{path:.*}", notFoundHandler)
	m.NotFoundHandler = notFoundHandler
	m.MethodNotAllowedHandler = notFoundHandler

	return m
}

func (server *Server) Start() {
	server.handleGracefulShutdown()

	log.Info().Str("address", server.httpAddress).Msg("Listening for HTTP on")
	list, err := net.Listen("tcp", server.httpAddress)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to open HTTP listener")
	}

	srv := &http.Server{Handler: server.router}
	server.mu.Lock()
	server.httpServer = srv
	server.mu.Unlock()
	err = srv.Serve(list)

	if err != http.ErrServerClosed {
		log.Fatal().Err(err).Msg("failed to serve HTTP server")
	}

}

func (server *Server) Stop() {
	server.mu.Lock()
	defer server.mu.Unlock()
	if server.httpServer != nil {
		server.httpServer.Close()
	}
}

func (server *Server) handleGracefulShutdown() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	go func() {
		sig := <-sigs

		log.Info().Interface("signal", sig).Msg("Server received signal, shutting down gracefully")
		server.Stop()
		os.Exit(0)
	}()
}
