package main

import (
	"context"
	"fmt"
	"os"
	"time"

	log "github.com/go-pkgz/lgr"
	"github.com/umputun/go-flags"

	"github.com/stsg/gophermart/cmd/gophermart/server"
	"github.com/stsg/gophermart/cmd/gophermart/service"
	postgres "github.com/stsg/gophermart/cmd/gophermart/store"
)

var opts struct {
	// Сервис должен поддерживать конфигурирование следующими методами:
	// адрес и порт запуска сервиса: переменная окружения ОС RUN_ADDRESS или флаг -a;
	// адрес подключения к базе данных: переменная окружения ОС DATABASE_URI или флаг -d;
	// адрес системы расчёта начислений: переменная окружения ОС ACCRUAL_SYSTEM_ADDRESS или флаг -r.
	RunAddr string `short:"a" long:"run-address" env:"RUN_ADDRESS" default:"localhost:8080" description:"server address"`
	DBURI   string `short:"d" long:"database-uri" env:"DATABASE_URI" default:"" description:"database uri"`
	AccAddr string `short:"r" long:"accrual-system-address" env:"ACCRUAL_SYSTEM_ADDRESS" default:"" description:"accrual system address"`
	Dbg     bool   `long:"dbg" description:"debug mode"`
}

var revision = "prototype-0.1.0"

func main() {
	if _, err := flags.Parse(&opts); err != nil {
		os.Exit(1)
	}
	fmt.Printf("gophermart %s\n", revision)

	setupLog(opts.Dbg)

	pCfg := &postgres.Config{
		ConnectionString: opts.DBURI,
		ConnectTimeout:   10 * time.Second,
		QueryTimeout:     10 * time.Second,
		MigrationVersion: 1,
	}

	storage, err := postgres.New(pCfg)
	if err != nil {
		log.Printf("[ERROR] DB connection error: %s", err)
		os.Exit(1)
	}

	srvc := service.New(storage, opts.AccAddr)
	go srvc.SendToAccrual(context.Background())
	go srvc.RecieveFromAccrual(context.Background())
	go srvc.ProcessOrders(context.Background())

	srv := server.Server{
		RunAddr: opts.RunAddr,
		AccAddr: opts.AccAddr,
		Service: srvc,
	}

	if err := srv.Run(context.Background()); err != nil {
		log.Printf("[ERROR] server failed, %v", err)
		os.Exit(1)
	}

}

func setupLog(dbg bool) {
	if dbg {
		log.Setup(log.Debug, log.CallerFile, log.Msec, log.LevelBraces)
		return
	}
	// log.Setup(log.Msec, log.LevelBraces)
	log.Setup(log.Debug, log.CallerFile, log.Msec, log.LevelBraces)
}
