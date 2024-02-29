package server

import (
	"context"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	log "github.com/go-pkgz/lgr"
	"github.com/jackc/pgerrcode"
	"github.com/pkg/errors"

	"github.com/stsg/gophermart/cmd/gophermart/lib"
	"github.com/stsg/gophermart/cmd/gophermart/models"
)

func (s Server) userRegisterCtrl(w http.ResponseWriter, r *http.Request) {
	var req models.UserRegisterRequest

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

	jwt, err := s.Service.Register(ctx, req.Login, req.Password)
	if err != nil {
		if errors.Is(err, models.ErrUserExists) {
			w.WriteHeader(http.StatusConflict)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	log.Printf("[INFO] login %s registered userRegisterCtrl", req.Login)
	w.Header().Set("Authorization", jwt)
	render.Status(r, http.StatusOK)
}

func (s Server) userLoginCtrl(w http.ResponseWriter, r *http.Request) {
	var req models.UserRegisterRequest

	ctx, cancel := context.WithTimeout(r.Context(), 1*time.Second)
	defer cancel()

	reqID := middleware.GetReqID(ctx)
	log.Printf("[INFO] reqID %s userLoginCtrl", reqID)

	err := render.DecodeJSON(r.Body, &req)
	if err != nil {
		log.Printf("[WARN] reqID %s userLoginCtrl, %v", reqID, err)
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, errors.Wrap(err, "failed to parse request body"))
		return
	}

	// TODO: case insensitive login
	// req.Login = strings.ToLower(req.Login)
	log.Printf("[INFO] login %s userLoginCtrl", req.Login)

	jwt, err := s.Service.Login(ctx, req.Login, req.Password)
	if err != nil {
		if errors.Is(err, models.ErrUserWrongPassword) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	log.Printf("[INFO] login %s authenticated userLoginCtrl", req.Login)
	w.Header().Set("Authorization", jwt)
	render.Status(r, http.StatusOK)
}

func (s Server) userPostOrdersCtrl(w http.ResponseWriter, r *http.Request) {
	var orderString string
	var orderNumber int64

	ctx, cancel := context.WithTimeout(r.Context(), 50*time.Second)
	defer cancel()

	reqID := middleware.GetReqID(ctx)
	log.Printf("[INFO] reqID %s userPostOrdersCtrl", reqID)

	user, ok := r.Context().Value(UserContextKey).(models.User)
	if !ok {
		render.Status(r, http.StatusUnauthorized)
		render.PlainText(w, r, "unauthorized\n")
		return
	}

	req, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("[ERROR] reqID %s userPostOrdersCtrl, %v", reqID, err)
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, errors.Wrap(err, "failed to parse request body"))
		return
	}

	orderString = string(req)
	orderNumber, err = strconv.ParseInt(string(req), 10, 64)
	if err != nil {
		log.Printf("[ERROR] reqID %s userPostOrdersCtrl, %v", reqID, err)
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, errors.Wrap(err, "cannot get order number"))
		return
	}

	if !lib.LuhnValid(orderNumber) {
		log.Printf("[ERROR] reqID %s userPostOrdersCtrl, %v", reqID, err)
		render.Status(r, http.StatusUnprocessableEntity)
		render.JSON(w, r, errors.Wrap(err, "invalid order number"))
		return
	}

	order, err := s.Service.SaveOrder(ctx, user.Login, orderString)
	if err != nil {
		log.Printf("[ERROR] reqID %s userPostOrdersCtrl, %v", reqID, err)
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, errors.Wrap(err, "cannot save order"))
		return
	}

	if order.UID != user.UID {
		log.Printf("[ERROR] reqID %s userPostOrdersCtrl, %v", reqID, err)
		render.Status(r, http.StatusConflict)
		render.JSON(w, r, errors.Wrap(err, "order uploaded by another user"))
		return
	}

	if order.AccrualStatus != models.AccrualStatusNew {
		log.Printf("[WARN] reqID %s userPostOrdersCtrl, %v", reqID, err)
		render.Status(r, http.StatusOK)
		render.JSON(w, r, errors.Wrap(err, "order uploaded by another user"))
		return
	}

	orderResponse := models.OrderResponse{
		ID:         orderString,
		Username:   user.Login,
		Status:     string(order.AccrualStatus),
		Amount:     order.Amount,
		UploadedAt: order.UploadedAt,
		// ProcessedAt: time.Time{},
	}

	log.Print("[INFO] trying to send to Accrual service")
	s.Service.ChanToAccurual <- &orderResponse
	log.Print("[INFO] sent to Accrual service")
	render.Status(r, http.StatusAccepted)
	render.JSON(w, r, "accepted")
}

func (s Server) userGetOrdersCtrl(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 50*time.Second)
	defer cancel()

	reqID := middleware.GetReqID(ctx)
	log.Printf("[INFO] reqID %s userGetOrdersCtrl", reqID)

	user, ok := r.Context().Value(UserContextKey).(models.User)
	if !ok {
		render.Status(r, http.StatusUnauthorized)
		render.PlainText(w, r, "unauthorized\n")
		return
	}

	orders, err := s.Service.GetOrders(ctx, user.Login)
	if err != nil {
		log.Printf("[ERROR] reqID %s userGetOrdersCtrl, %v", reqID, err)
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, errors.Wrap(err, "cannot get orders"))
		return
	}

	if len(orders) == 0 {
		log.Printf("[INFO] reqID %s userGetOrdersCtrl, %v", reqID, err)
		render.Status(r, http.StatusNoContent)
		render.JSON(w, r, "no orders")
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, orders)
}

func (s Server) userBalanceCtrl(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 50*time.Second)
	defer cancel()

	reqID := middleware.GetReqID(ctx)
	log.Printf("[INFO] reqID %s userBalanceCtrl", reqID)

	user, ok := r.Context().Value(UserContextKey).(models.User)
	if !ok {
		render.Status(r, http.StatusUnauthorized)
		render.PlainText(w, r, "unauthorized\n")
		return
	}

	balance, err := s.Service.GetBalance(ctx, user.Login)
	if err != nil {
		log.Printf("[ERROR] reqID %s userBalanceCtrl, %v", reqID, err)
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, errors.Wrap(err, "cannot get balance"))
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, balance)
}

func (s Server) userWithdrawCtrl(w http.ResponseWriter, r *http.Request) {
	var req models.WithdrawRequest
	// var res models.WithdrawResponse

	ctx, cancel := context.WithTimeout(r.Context(), 50*time.Second)
	defer cancel()

	reqID := middleware.GetReqID(ctx)
	log.Printf("[INFO] reqID %s userWithdrawCtrl", reqID)

	user, ok := r.Context().Value(UserContextKey).(models.User)
	if !ok {
		render.Status(r, http.StatusUnauthorized)
		render.PlainText(w, r, "unauthorized\n")
		return
	}

	err := render.DecodeJSON(r.Body, &req)
	if err != nil {
		log.Printf("[WARN] reqID %s userWithdrawCtrl, %v", reqID, err)
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, errors.Wrap(err, "failed to parse request body"))
		return
	}

	orderNumber, err := strconv.ParseInt(req.Number, 10, 64)
	if err != nil {
		log.Printf("[WARN] reqID %s userWithdrawCtrl, %v", reqID, err)
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, errors.Wrap(err, "cannot get witdraw number"))
		return
	}

	if !lib.LuhnValid(orderNumber) {
		log.Printf("[ERROR] reqID %s userWithdrawCtrl, %v", reqID, err)
		render.Status(r, http.StatusUnprocessableEntity)
		render.JSON(w, r, errors.Wrap(err, "invalid order number"))
		return
	}

	err = s.Service.SaveWithdraw(ctx, user.Login, req.Number, req.Accrual)

	if errors.As(err, pgerrcode.UniqueViolation) {
		log.Printf("[ERROR] reqID %s userWithdrawCtrl, %v", reqID, err)
		render.Status(r, http.StatusUnprocessableEntity)
		render.JSON(w, r, errors.Wrap(err, "duplicate order number"))
		return
	}

	if errors.As(err, models.ErrBalanceWrong) {
		log.Printf("[ERROR] reqID %s userWithdrawCtrl, %v", reqID, err)
		render.Status(r, http.StatusPaymentRequired)
		render.JSON(w, r, errors.Wrap(err, "not enough money in the account"))
		return
	}

	res := models.WithdrawResponse{
		Number:      req.Number,
		Accrual:     int(req.Accrual),
		ProcessedAt: time.Now(),
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, res)
}

func (s Server) userGetWithdrawalsCtrl(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 50*time.Second)
	defer cancel()

	reqID := middleware.GetReqID(ctx)
	log.Printf("[INFO] reqID %s userWithdrawalsCtrl", reqID)

	user, ok := r.Context().Value(UserContextKey).(models.User)
	if !ok {
		render.Status(r, http.StatusUnauthorized)
		render.PlainText(w, r, "unauthorized\n")
		return
	}

	// err := render.DecodeJSON(r.Body, &req)
	// if err != nil {
	// 	log.Printf("[WARN] reqID %s userWithdrawalsCtrl, %v", reqID, err)
	// 	render.Status(r, http.StatusBadRequest)
	// 	render.JSON(w, r, errors.Wrap(err, "failed to parse request body"))
	// 	return
	// }

	withdrawals, err := s.Service.GetWithdrawals(ctx, user.Login)
	if err != nil {
		log.Printf("[ERROR] reqID %s userWithdrawalsCtrl, %v", reqID, err)
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, errors.Wrap(err, "cannot get withdrawals"))
		return
	}

	if len(withdrawals) == 0 {
		log.Printf("[ERROR] reqID %s userWithdrawalsCtrl, no withdrawals", reqID)
		render.Status(r, http.StatusNoContent)
		render.JSON(w, r, "cannot get withdrawals")
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, withdrawals)
}
