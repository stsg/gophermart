package main

import (
	"fmt"
	"os"

	log "github.com/go-pkgz/lgr"
	"github.com/umputun/go-flags"

	"github.com/stsg/gophermart/store"
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

var revision = "dev-0.1.0"

func main() {
	if _, err := flags.Parse(&opts); err != nil {
		os.Exit(1)
	}
	fmt.Printf("gophermart %s\n", revision)

	setupLog(opts.Dbg)

	store := store.NewStore(opts.DBURI)
	srv := NewServer(store, opts.RunAddr, opts.AccAddr)
	defer srv.Close()

}

func setupLog(dbg bool) {
	if dbg {
		log.Setup(log.Debug, log.CallerFile, log.Msec, log.LevelBraces)
		return
	}
	log.Setup(log.Msec, log.LevelBraces)
}
