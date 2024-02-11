package server

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	log "github.com/go-pkgz/lgr"
	"github.com/pkg/errors"

	"github.com/go-pkgz/rest"
)

type Server struct {
	store   Store
	runAddr string
	accAddr string
}

func (s Server) Run(ctx context.Context) error {
	log.Printf("[INFO] activate rest server")

	httpServer := &http.Server{
		Addr:              s.runAddr,
		Handler:           s.routes(),
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       30 * time.Second,
	}

	go func() {
		<-ctx.Done()
		if httpServer != nil {
			if clsErr := httpServer.Close(); clsErr != nil {
				log.Printf("[ERROR] failed to close proxy http server, %v", clsErr)
			}
		}
	}()

	err := httpServer.ListenAndServe()
	log.Printf("[WARN] http server terminated, %s", err)

	if !errors.Is(err, http.ErrServerClosed) {
		return errors.Wrap(err, "server failed")
	}
	return nil
}

func (s Server) routes() chi.Router {
	router := chi.NewRouter()

	router.Use(middleware.RequestID, middleware.RealIP, rest.Recoverer(log.Default()))
	router.Use(middleware.Throttle(1000), middleware.Timeout(60*time.Second))

	router.Get("/ping", s.getPing)
	router.Route("/api", func(r chi.Router) {
		r.Use(Logger(log.Default()))
		r.Post("/user/register", s.userRegisterCtrl)
		r.Post("/user/login", s.userLoginCtrl)
		r.Post("/user/orders", s.userOrdersCtrl)
		r.Get("/user/orders", s.userGetOrdersCtrl)
		r.Get("/user/balance", s.userBalanceCtrl)
		r.Post("/user/balance/withdraw", s.userWithdrawCtrl)
		r.Get("/user/balance/withdrawals", s.userGetWithdrawalsCtrl)
	})

	return router
}

func (s Server) getPing(w http.ResponseWriter, r *http.Request) {
	render.Status(r, http.StatusOK)
	render.PlainText(w, r, "pong\n")
}

func (s Server) userRegisterCtrl(w http.ResponseWriter, r *http.Request) {
	render.Status(r, http.StatusOK)
	render.PlainText(w, r, "pong\n")
}

func (s Server) userLoginCtrl(w http.ResponseWriter, r *http.Request) {
	render.Status(r, http.StatusOK)
	render.PlainText(w, r, "pong\n")
}

func (s Server) userOrdersCtrl(w http.ResponseWriter, r *http.Request) {
	render.Status(r, http.StatusOK)
	render.PlainText(w, r, "pong\n")
}

func (s Server) userGetOrdersCtrl(w http.ResponseWriter, r *http.Request) {
	render.Status(r, http.StatusOK)
	render.PlainText(w, r, "pong\n")
}

func (s Server) userBalanceCtrl(w http.ResponseWriter, r *http.Request) {
	render.Status(r, http.StatusOK)
	render.PlainText(w, r, "pong\n")
}

func (s Server) userWithdrawCtrl(w http.ResponseWriter, r *http.Request) {
	render.Status(r, http.StatusOK)
	render.PlainText(w, r, "pong\n")
}

func (s Server) userGetWithdrawalsCtrl(w http.ResponseWriter, r *http.Request) {
	render.Status(r, http.StatusOK)
	render.PlainText(w, r, "pong\n")
}
