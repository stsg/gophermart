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

	"github.com/stsg/gophermart/cmd/gophermart/store"
)

type Server struct {
	Store   store.Store
	RunAddr string
	AccAddr string
}

type UserRegisterRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
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
	var req UserRegisterRequest

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	reqID := middleware.GetReqID(ctx)
	log.Printf("[INFO] reqID %s userRegisterCtrl", reqID)

	err := render.DecodeJSON(r.Body, &req)
	if err != nil {
		log.Printf("[WARN] reqID %s userRegisterCtrl, %v", reqID, err)
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, errors.Wrap(err, "failed to parse request body"))
		return
	}

	log.Printf("[INFO] login %s userRegisterCtrl", req.Login)

	// TODO
	// registration code

	log.Printf("[INFO] logini %s registered userRegisterCtrl", req.Login)
	w.Header().Set("Authorization", "Bearer "+req.Login)
	render.Status(r, http.StatusOK)
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
