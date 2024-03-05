package server

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	log "github.com/go-pkgz/lgr"
	"github.com/go-pkgz/rest"
	"github.com/pkg/errors"

	"github.com/stsg/gophermart/cmd/gophermart/service"
)

type Server struct {
	RunAddr string
	AccAddr string
	Service *service.Service
}

func (s Server) Run(ctx context.Context) error {
	log.Printf("[INFO] activate server")

	httpServer := &http.Server{
		Addr:              s.RunAddr,
		Handler:           s.routes(),
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       30 * time.Second,
	}

	go func() {
		<-ctx.Done()
		if httpServer != nil {
			if clsErr := httpServer.Close(); clsErr != nil {
				log.Printf("[ERROR] failed to close http server, %v", clsErr)
			}
		}
	}()

	err := httpServer.ListenAndServe()
	log.Printf("[WARN] server terminated, %s", err)

	if !errors.Is(err, http.ErrServerClosed) {
		return errors.Wrap(err, "server failed")
	}
	return nil
}

func (s Server) routes() chi.Router {
	router := chi.NewRouter()

	router.Use(middleware.RequestID, middleware.RealIP, rest.Recoverer(log.Default()))
	router.Use(middleware.Throttle(1000), middleware.Timeout(60*time.Second))
	router.Use(middleware.Compress(5, "application/json", "text/html"))
	router.Use(Decompress())

	router.Get("/ping", s.getPing)
	router.Route("/api", func(r chi.Router) {
		r.Use(Logger(log.Default()))
		r.Post("/user/register", s.userRegisterCtrl)
		r.Post("/user/login", s.userLoginCtrl)
		r.Group(func(r chi.Router) {
			r.Use(Authorize(s.Service))
			r.Post("/user/orders", s.userPostOrdersCtrl)
			r.Get("/user/orders", s.userGetOrdersCtrl)
			r.Get("/user/balance", s.userBalanceCtrl)
			r.Post("/user/balance/withdraw", s.userWithdrawCtrl)
			r.Get("/user/withdrawals", s.userGetWithdrawalsCtrl)
		})
	})

	return router
}

func (s Server) getPing(w http.ResponseWriter, r *http.Request) {
	render.Status(r, http.StatusOK)
	render.PlainText(w, r, "pong\n")
}
