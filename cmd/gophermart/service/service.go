// service, that implements business logic of gophermart

package service

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	log "github.com/go-pkgz/lgr"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/stsg/gophermart/cmd/gophermart/lib"
	"github.com/stsg/gophermart/cmd/gophermart/models"
	postgres "github.com/stsg/gophermart/cmd/gophermart/store"
)

// type Service interface {
// 	GetUserByLogin(ctx context.Context, login string) (*models.User, error)
// 	Login(ctx context.Context, login, password string) (string, error)
// 	PostOrders(ctx context.Context, login string, orderNum int64) error
// 	GetOrders(ctx context.Context, login string) error
// 	GetBalance(ctx context.Context, login string) (int64, error)
// 	Withdraw(ctx context.Context, login string) error
// 	GetWithdrawals(ctx context.Context, login string) error
// }

type Service struct {
	storage          *postgres.Storage
	ChanToAccurual   chan *models.OrderResponse
	ChanFromAccurual chan *models.OrderResponse
	accrualAddress   string
}

func New(storage *postgres.Storage, accrualAddress string) *Service {
	toAccurual := make(chan *models.OrderResponse, 100)
	fromAccurual := make(chan *models.OrderResponse, 100)

	return &Service{
		storage:          storage,
		ChanToAccurual:   toAccurual,
		ChanFromAccurual: fromAccurual,
		accrualAddress:   accrualAddress,
	}
}

func (s *Service) RegisterNewOrder(ctx context.Context) {
	log.Printf("[INFO] registerNewOrder")
	for {
		log.Print("[INFO] waiting for ChanToAccurual")
		order := <-s.ChanToAccurual
		log.Print("[INFO] received from ChanToAccurual")

		url, err := url.Parse(s.accrualAddress + "/api/orders")
		if err != nil {
			log.Printf("[ERROR] accrualAddress invalid %v", err)
		}

		body := fmt.Sprintf(`{"order": "%s"}`, order.Number)
		jsonBody := []byte(body)
		req := bytes.NewReader(jsonBody)
		resp, err := http.Post(url.String(), "application/json", req)
		if err != nil {
			s.ChanToAccurual <- order
			log.Printf("[ERROR] cant get accrual %v", err)
		}

		if resp.StatusCode == http.StatusAccepted || resp.StatusCode == http.StatusConflict {
			order, _ = s.storage.UpdateOrderStatus(ctx, order.Number, models.AccrualStatusProcessed)
			log.Print("[INFO] trying sending to ChanFromAccurual")
			s.ChanFromAccurual <- order
			log.Print("[INFO] sent to ChanFromAccurual")
		} else {
			log.Print("[INFO] trying sending ChanToAccurual")
			s.ChanToAccurual <- order
			log.Print("[INFO] sent to ChanToAccurual")

		}
		err = resp.Body.Close()
		if err != nil {
			log.Printf("[ERROR] cant close request body %v", err)
		}
	}
}

func (s *Service) GetUserByLogin(ctx context.Context, login string) (*models.User, error) {

	user, err := s.storage.GetUserByLogin(ctx, login)

	if err != nil {
		log.Printf("[ERROR] user %s not found %v", user.Login, err)
		return nil, models.ErrUserNotFound
	}

	return user, nil
}

func (s *Service) Login(ctx context.Context, login, password string) (string, error) {

	user, err := s.GetUserByLogin(ctx, login)
	if err != nil {
		return "", err
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PHash), []byte(password))
	if err != nil {
		log.Printf("[ERROR] user %s wrong password %v", login, err)
		return "", models.ErrUserWrongPassword
	}

	jwtString, err := lib.CreateJWT(user.UUID)
	if err != nil {
		log.Printf("[ERROR] failed to create JWT %v", err)
		return "", err
	}

	return jwtString, nil
}

// TODO should be renamed to Login
func (s *Service) GetUserByToken(ctx context.Context, token string) (models.User, error) {

	userUUID, err := lib.CheckJWT(token)
	if err != nil || userUUID == uuid.Nil {
		log.Printf("[ERROR] invalid JWT %s", err)
		return models.User{}, err
	}

	user, err := s.storage.GetUserByUUID(ctx, userUUID)
	if err != nil {
		log.Printf("[ERROR] user %s not found %v", user.Login, err)
		return models.User{}, models.ErrUserNotFound
	}

	return user, nil
}

func (s *Service) Register(ctx context.Context, login, password string) (string, error) {
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("[ERROR] cannot generate passowrd hash %v", err)
		return "", err
	}

	user := &models.User{
		Login:    login,
		PHash:    string(passwordHash),
		UUID:     uuid.New(),
		JWTToken: "",
	}

	user, err = s.storage.CreateUser(ctx, user)
	if err != nil {
		return "", err
	}

	jwtString, err := lib.CreateJWT(user.UUID)
	if err != nil {
		log.Printf("[ERROR] failed to create JWT %v", err)
		return "", err
	}

	return jwtString, nil
}

func (s *Service) SaveOrder(ctx context.Context, login string, orderNum int64) (order *models.Order, err error) {
	user, err := s.storage.GetUserByLogin(ctx, login)
	if err != nil {
		log.Printf("[ERROR] user %s not found %v", user.Login, err)
		return nil, models.ErrUserNotFound
	}

	order, err = s.storage.SaveOrder(ctx, user, &models.Order{
		// ID:            string(rune(orderNum)),
		ID:            strconv.FormatInt(orderNum, 10),
		UUID:          user.UUID,
		Accrual:       0,
		AccrualStatus: models.AccrualStatusNew,
		UploadedAt:    time.Now(),
	})
	if err != nil {
		log.Printf("[ERROR] cannot save order %s %v", user.Login, err)
		return nil, err
	}

	return order, nil
}
