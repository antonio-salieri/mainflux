// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/BurntSushi/toml"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/jmoiron/sqlx"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/internal/pkg/server"
	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/transformers/senml"
	"github.com/mainflux/mainflux/writers"
	"github.com/mainflux/mainflux/writers/api"
	"github.com/mainflux/mainflux/writers/postgres"
	nats "github.com/nats-io/go-nats"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
)

const (
	svcName = "postgres-writer"
	sep     = ","

	defNatsURL       = nats.DefaultURL
	defLogLevel      = "error"
	defPort          = "9104"
	defDBHost        = "postgres"
	defDBPort        = "5432"
	defDBUser        = "mainflux"
	defDBPass        = "mainflux"
	defDBName        = "messages"
	defDBSSLMode     = "disable"
	defDBSSLCert     = ""
	defDBSSLKey      = ""
	defDBSSLRootCert = ""
	defChanCfgPath   = "/config/channels.toml"

	envNatsURL       = "MF_NATS_URL"
	envLogLevel      = "MF_POSTGRES_WRITER_LOG_LEVEL"
	envPort          = "MF_POSTGRES_WRITER_PORT"
	envDBHost        = "MF_POSTGRES_WRITER_DB_HOST"
	envDBPort        = "MF_POSTGRES_WRITER_DB_PORT"
	envDBUser        = "MF_POSTGRES_WRITER_DB_USER"
	envDBPass        = "MF_POSTGRES_WRITER_DB_PASS"
	envDBName        = "MF_POSTGRES_WRITER_DB_NAME"
	envDBSSLMode     = "MF_POSTGRES_WRITER_DB_SSL_MODE"
	envDBSSLCert     = "MF_POSTGRES_WRITER_DB_SSL_CERT"
	envDBSSLKey      = "MF_POSTGRES_WRITER_DB_SSL_KEY"
	envDBSSLRootCert = "MF_POSTGRES_WRITER_DB_SSL_ROOT_CERT"
	envChanCfgPath   = "MF_POSTGRES_WRITER_CHANNELS_CONFIG"
)

type config struct {
	natsURL  string
	logLevel string
	port     string
	dbConfig postgres.Config
	channels map[string]bool
}

func main() {
	cfg := loadConfig()

	logger, err := logger.New(os.Stdout, cfg.logLevel)
	if err != nil {
		log.Fatalf(err.Error())
	}

	nc := connectToNATS(cfg.natsURL, logger)
	defer nc.Close()

	db := connectToDB(cfg.dbConfig, logger)
	defer db.Close()

	repo := newService(db, logger)
	st := senml.New()
	// TODO: consider adding graceful Stop method
	if err = writers.Start(nc, repo, st, svcName, cfg.channels, logger); err != nil {
		logger.Error(fmt.Sprintf("Failed to create Postgres writer: %s", err))
	}

	errs := make(chan error, 2)

	httpServer := server.NewHTTPServer(fmt.Sprintf(":%s", cfg.port), api.MakeHandler(svcName), "", "")
	go httpServer.Start(logger, errs)

	server.Monitor(logger, errs, httpServer)

	logger.Info("Postgres writer service terminated")
}

func loadConfig() config {
	chanCfgPath := mainflux.Env(envChanCfgPath, defChanCfgPath)
	dbConfig := postgres.Config{
		Host:        mainflux.Env(envDBHost, defDBHost),
		Port:        mainflux.Env(envDBPort, defDBPort),
		User:        mainflux.Env(envDBUser, defDBUser),
		Pass:        mainflux.Env(envDBPass, defDBPass),
		Name:        mainflux.Env(envDBName, defDBName),
		SSLMode:     mainflux.Env(envDBSSLMode, defDBSSLMode),
		SSLCert:     mainflux.Env(envDBSSLCert, defDBSSLCert),
		SSLKey:      mainflux.Env(envDBSSLKey, defDBSSLKey),
		SSLRootCert: mainflux.Env(envDBSSLRootCert, defDBSSLRootCert),
	}

	return config{
		natsURL:  mainflux.Env(envNatsURL, defNatsURL),
		logLevel: mainflux.Env(envLogLevel, defLogLevel),
		port:     mainflux.Env(envPort, defPort),
		dbConfig: dbConfig,
		channels: loadChansConfig(chanCfgPath),
	}
}

type channels struct {
	List []string `toml:"filter"`
}

type chanConfig struct {
	Channels channels `toml:"channels"`
}

func loadChansConfig(chanConfigPath string) map[string]bool {
	data, err := ioutil.ReadFile(chanConfigPath)
	if err != nil {
		log.Fatal(err)
	}

	var chanCfg chanConfig
	if err := toml.Unmarshal(data, &chanCfg); err != nil {
		log.Fatal(err)
	}

	chans := map[string]bool{}
	for _, ch := range chanCfg.Channels.List {
		chans[ch] = true
	}

	return chans
}

func connectToNATS(url string, logger logger.Logger) *nats.Conn {
	nc, err := nats.Connect(url)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to NATS: %s", err))
		os.Exit(1)
	}

	return nc
}

func connectToDB(dbConfig postgres.Config, logger logger.Logger) *sqlx.DB {
	db, err := postgres.Connect(dbConfig)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to Postgres: %s", err))
		os.Exit(1)
	}
	return db
}

func newService(db *sqlx.DB, logger logger.Logger) writers.MessageRepository {
	svc := postgres.New(db)
	svc = api.LoggingMiddleware(svc, logger)
	svc = api.MetricsMiddleware(
		svc,
		kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace: "postgres",
			Subsystem: "message_writer",
			Name:      "request_count",
			Help:      "Number of requests received.",
		}, []string{"method"}),
		kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
			Namespace: "postgres",
			Subsystem: "message_writer",
			Name:      "request_latency_microseconds",
			Help:      "Total duration of requests in microseconds.",
		}, []string{"method"}),
	)

	return svc
}
