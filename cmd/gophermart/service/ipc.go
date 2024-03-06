package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	log "github.com/go-pkgz/lgr"

	"github.com/stsg/gophermart/cmd/gophermart/models"
)

func (s *Service) SendToAccrual(ctx context.Context) {
	log.Printf("[INFO] SendToAccrual")
	for {
		log.Print("[INFO] waiting for ChanToAccurual")
		order := <-s.ChanToAccurual
		log.Print("[INFO] received from ChanToAccurual")

		// url, err := url.Parse("http://" + s.accrualAddress + "/api/orders")
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
			log.Printf("[ERROR] cant get answer from accrual %v", err)
		}

		if resp.StatusCode == http.StatusAccepted || resp.StatusCode == http.StatusConflict {
			order, _ := s.storage.UpdateOrderStatus(ctx, order.ID, models.AccrualStatusProcessed, int64(order.Amount*100))
			log.Print("[INFO] trying sending to ChanFromAccurual")
			s.ChanFromAccurual <- &order
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
		order := <-s.ChanFromAccurual
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

		if resp.StatusCode == http.StatusOK && accrual.Status == models.AccrualStatusProcessed {
			_, err := s.storage.UpdateOrderStatus(ctx, order.ID, models.AccrualStatusProcessed, int64(accrual.Accrual*100.00))
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

func (s *Service) ProcessOrders(ctx context.Context) {
	log.Printf("[INFO] ProcessOrders")

	newOrders, err := s.storage.GetOrdersByStatus(ctx, models.AccrualStatusNew)
	if err != nil {
		log.Printf("[ERROR] cannot get new orders %v", err)
	}

	processingOrders, err := s.storage.GetOrdersByStatus(ctx, models.AccrualStatusProcessing)
	if err != nil {
		log.Printf("[ERROR] cannot get processing orders %v", err)
	}

	for _, order := range newOrders {
		s.ChanToAccurual <- &order
	}

	for _, order := range processingOrders {
		s.ChanFromAccurual <- &order
	}
}
