package service

import (
	"context"
	"time"

	log "github.com/go-pkgz/lgr"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/stsg/gophermart/cmd/gophermart/lib"
	"github.com/stsg/gophermart/cmd/gophermart/models"
	postgres "github.com/stsg/gophermart/cmd/gophermart/store"
)

// ---------------------------------8<-----------------------------------
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

func (s *Service) SaveWithdraw(ctx context.Context, login string, orderNum string, amount int64) (err error) {
	user, err := s.storage.GetUserByLogin(ctx, login)
	if err != nil {
		log.Printf("[ERROR] user %s not found %v", user.Login, err)
		return models.ErrUserNotFound
	}

	err = s.storage.SaveWithdraw(ctx, user, &models.Order{
		ID:            orderNum,
		UID:           user.UID,
		Amount:        amount,
		AccrualStatus: models.AccrualStatusNew,
		UploadedAt:    time.Now(),
	})
	if err != nil {
		log.Printf("[ERROR] cannot save withdraw %s %v", user.Login, err)
		return err
	}

	return err
}

func (s *Service) GetWithdrawals(ctx context.Context, login string) ([]models.WithdrawalsResponse, error) {
	user, err := s.storage.GetUserByLogin(ctx, login)
	if err != nil {
		log.Printf("[ERROR] user %s not found %v", user.Login, err)
		return nil, models.ErrUserNotFound
	}

	return s.storage.GetWithdrawals(ctx, user.UID)
}
