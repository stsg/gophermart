// service, that implements business logic of gophermart

package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	log "github.com/go-pkgz/lgr"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/stsg/gophermart/cmd/gophermart/lib"
	"github.com/stsg/gophermart/cmd/gophermart/models"
	postgres "github.com/stsg/gophermart/cmd/gophermart/store"
)

// ---------------------------------8<-----------------------------------
// type Service interface {
// 	GetUserByLogin(ctx context.Context, login string) (*models.User, error)
// 	Login(ctx context.Context, login, password string) (string, error)
// 	PostOrders(ctx context.Context, login string, orderNum int64) error
// 	GetOrders(ctx context.Context, login string) error
// 	GetBalance(ctx context.Context, login string) (int64, error)
// 	Withdraw(ctx context.Context, login string) error
// 	GetWithdrawals(ctx context.Context, login string) error
// }
// --------------------------------->8-----------------------------------

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

func (s *Service) SendToAccrual(ctx context.Context) {
	log.Printf("[INFO] SendToAccrual")
	for {
		log.Print("[INFO] waiting for ChanToAccurual")
		order := <-s.ChanToAccurual
		log.Print("[INFO] received from ChanToAccurual")

		url, err := url.Parse(s.accrualAddress + "/api/orders")
		if err != nil {
			log.Printf("[ERROR] accrualAddress invalid %v", err)
		}

		body := fmt.Sprintf(`{"order": "%s"}`, order.ID)
		jsonBody := []byte(body)
		req := bytes.NewReader(jsonBody)
		resp, err := http.Post(url.String(), "application/json", req)
		if err != nil {
			s.ChanToAccurual <- order
			log.Printf("[ERROR] cant get accrual %v", err)
		}

		if resp.StatusCode == http.StatusAccepted || resp.StatusCode == http.StatusConflict {
			order, _ = s.storage.UpdateOrderStatus(ctx, order.ID, models.AccrualStatusProcessed, 0)
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

func (s *Service) RecieveFromAccrual(ctx context.Context) {
	log.Printf("[INFO] RecieveFromAccrual")
	for {
		log.Print("[INFO] waiting for ChanFromAccurual")
		order := <-s.ChanToAccurual
		log.Print("[INFO] received from ChanFromAccurual")
		url, err := url.Parse(fmt.Sprintf(s.accrualAddress+"/api/orders/%s", order.ID))
		if err != nil {
			log.Printf("[ERROR] accrualAddress invalid %v", err)
		}

		resp, err := http.Get(url.String())
		if err != nil {
			log.Printf("[ERROR] cant get accrual %v", err)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Printf("[ERROR] cannot get body %v", err)
		}

		accrual := &models.AccrualResponse{}
		err = json.Unmarshal(body, accrual)
		if err != nil {
			log.Printf("[ERROR] cannot unmarshal body %v", err)
		}

		if resp.StatusCode == http.StatusOK && accrual.Status == string(models.AccrualStatusProcessed) {
			_, err := s.storage.UpdateOrderStatus(ctx, order.ID, models.AccrualStatusProcessed, accrual.Accrual)
			if err != nil {
				return
			}
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

	jwtString, err := lib.CreateJWT(user.UID)
	if err != nil {
		log.Printf("[ERROR] failed to create JWT %v", err)
		return "", err
	}

	return jwtString, nil
}

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
		UID:      uuid.New(),
		JWTToken: "",
	}

	user, err = s.storage.CreateUser(ctx, user)
	if err != nil {
		return "", err
	}

	jwtString, err := lib.CreateJWT(user.UID)
	if err != nil {
		log.Printf("[ERROR] failed to create JWT %v", err)
		return "", err
	}

	return jwtString, nil
}

func (s *Service) SaveOrder(ctx context.Context, login string, orderNum string) (order *models.Order, err error) {
	user, err := s.storage.GetUserByLogin(ctx, login)
	if err != nil {
		log.Printf("[ERROR] user %s not found %v", user.Login, err)
		return nil, models.ErrUserNotFound
	}

	order, err = s.storage.SaveOrder(ctx, user, &models.Order{
		ID:            orderNum,
		UID:           user.UID,
		Amount:        0,
		AccrualStatus: models.AccrualStatusNew,
		UploadedAt:    time.Now(),
	})
	if err != nil {
		log.Printf("[ERROR] cannot save order %s %v", user.Login, err)
		return nil, err
	}

	return order, nil
}

func (s *Service) GetOrders(ctx context.Context, login string) ([]models.OrderResponse, error) {
	user, err := s.storage.GetUserByLogin(ctx, login)
	if err != nil {
		log.Printf("[ERROR] user %s not found %v", user.Login, err)
		return nil, models.ErrUserNotFound
	}
	return s.storage.GetOrders(ctx, user.UID), nil
}

func (s *Service) GetBalance(ctx context.Context, login string) (models.BalanceResponse, error) {
	user, err := s.storage.GetUserByLogin(ctx, login)
	if err != nil {
		log.Printf("[ERROR] user %s not found %v", user.Login, err)
		return models.BalanceResponse{}, models.ErrUserNotFound
	}
	return s.storage.GetBalance(ctx, user.UID), nil
}

func (s *Service) SaveWithdraw(ctx context.Context, login string, orderNum string, accrual int64) (order *models.Order, err error) {
	user, err := s.storage.GetUserByLogin(ctx, login)
	if err != nil {
		log.Printf("[ERROR] user %s not found %v", user.Login, err)
		return nil, models.ErrUserNotFound
	}

	order, err = s.storage.SaveWithdraw(ctx, user, &models.Order{
		ID:            orderNum,
		UID:           user.UID,
		Amount:        accrual,
		AccrualStatus: models.AccrualStatusNew,
		UploadedAt:    time.Now(),
	})
	if err != nil {
		log.Printf("[ERROR] cannot save withdraw %s %v", user.Login, err)
		return nil, err
	}

	return order, nil
}
