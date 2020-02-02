package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"

	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/jmoiron/sqlx"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/authn"
	api "github.com/mainflux/mainflux/authn/api"
	grpcapi "github.com/mainflux/mainflux/authn/api/grpc"
	httpapi "github.com/mainflux/mainflux/authn/api/http"
	"github.com/mainflux/mainflux/authn/jwt"
	"github.com/mainflux/mainflux/authn/postgres"
	"github.com/mainflux/mainflux/authn/tracing"
	mfidp "github.com/mainflux/mainflux/authn/uuid"
	"github.com/mainflux/mainflux/internal/pkg/server"
	"github.com/mainflux/mainflux/logger"
	"github.com/opentracing/opentracing-go"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	jconfig "github.com/uber/jaeger-client-go/config"
)

const (
	defLogLevel      = "error"
	defDBHost        = "localhost"
	defDBPort        = "5432"
	defDBUser        = "mainflux"
	defDBPass        = "mainflux"
	defDBName        = "authn"
	defDBSSLMode     = "disable"
	defDBSSLCert     = ""
	defDBSSLKey      = ""
	defDBSSLRootCert = ""
	defHTTPPort      = "8180"
	defGRPCPort      = "8181"
	defSecret        = "authn"
	defServerCert    = ""
	defServerKey     = ""
	defJaegerURL     = ""

	envLogLevel      = "MF_AUTHN_LOG_LEVEL"
	envDBHost        = "MF_AUTHN_DB_HOST"
	envDBPort        = "MF_AUTHN_DB_PORT"
	envDBUser        = "MF_AUTHN_DB_USER"
	envDBPass        = "MF_AUTHN_DB_PASS"
	envDBName        = "MF_AUTHN_DB"
	envDBSSLMode     = "MF_AUTHN_DB_SSL_MODE"
	envDBSSLCert     = "MF_AUTHN_DB_SSL_CERT"
	envDBSSLKey      = "MF_AUTHN_DB_SSL_KEY"
	envDBSSLRootCert = "MF_AUTHN_DB_SSL_ROOT_CERT"
	envHTTPPort      = "MF_AUTHN_HTTP_PORT"
	envGRPCPort      = "MF_AUTHN_GRPC_PORT"
	envSecret        = "MF_AUTHN_SECRET"
	envServerCert    = "MF_AUTHN_SERVER_CERT"
	envServerKey     = "MF_AUTHN_SERVER_KEY"
	envJaegerURL     = "MF_JAEGER_URL"
)

type config struct {
	logLevel   string
	dbConfig   postgres.Config
	httpPort   string
	grpcPort   string
	secret     string
	serverCert string
	serverKey  string
	jaegerURL  string
	resetURL   string
}

type tokenConfig struct {
	hmacSampleSecret []byte // secret for signing token
	tokenDuration    string // token in duration in min
}

func main() {
	cfg := loadConfig()

	logger, err := logger.New(os.Stdout, cfg.logLevel)
	if err != nil {
		log.Fatalf(err.Error())
	}

	db := connectToDB(cfg.dbConfig, logger)
	defer db.Close()

	tracer, closer := initJaeger("authn", cfg.jaegerURL, logger)
	defer closer.Close()

	dbTracer, dbCloser := initJaeger("authn_db", cfg.jaegerURL, logger)
	defer dbCloser.Close()

	svc := newService(db, dbTracer, cfg.secret, logger)
	errs := make(chan error, 2)

	httpServer := server.NewHTTPServer(
		fmt.Sprintf(":%s", cfg.httpPort),
		httpapi.MakeHandler(svc, tracer),
		cfg.serverCert,
		cfg.serverKey)
	go httpServer.Start(logger, errs)

	grpcServer := server.NewGRPCServer(
		fmt.Sprintf(":%s", cfg.grpcPort),
		cfg.serverCert,
		cfg.serverKey,
		logger,
	)
	go grpcServer.Start(logger, errs, func() { mainflux.RegisterAuthNServiceServer(grpcServer.Server, grpcapi.NewServer(tracer, svc)) })

	server.Monitor(logger, errs, httpServer, grpcServer)

	logger.Info("Authentication service terminated")
}

func loadConfig() config {
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
		logLevel:   mainflux.Env(envLogLevel, defLogLevel),
		dbConfig:   dbConfig,
		httpPort:   mainflux.Env(envHTTPPort, defHTTPPort),
		grpcPort:   mainflux.Env(envGRPCPort, defGRPCPort),
		secret:     mainflux.Env(envSecret, defSecret),
		serverCert: mainflux.Env(envServerCert, defServerCert),
		serverKey:  mainflux.Env(envServerKey, defServerKey),
		jaegerURL:  mainflux.Env(envJaegerURL, defJaegerURL),
	}

}

func initJaeger(svcName, url string, logger logger.Logger) (opentracing.Tracer, io.Closer) {
	if url == "" {
		return opentracing.NoopTracer{}, ioutil.NopCloser(nil)
	}

	tracer, closer, err := jconfig.Configuration{
		ServiceName: svcName,
		Sampler: &jconfig.SamplerConfig{
			Type:  "const",
			Param: 1,
		},
		Reporter: &jconfig.ReporterConfig{
			LocalAgentHostPort: url,
			LogSpans:           true,
		},
	}.NewTracer()
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to init Jaeger: %s", err))
		os.Exit(1)
	}

	return tracer, closer
}

func connectToDB(dbConfig postgres.Config, logger logger.Logger) *sqlx.DB {
	db, err := postgres.Connect(dbConfig)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to postgres: %s", err))
		os.Exit(1)
	}
	return db
}

func newService(db *sqlx.DB, tracer opentracing.Tracer, secret string, logger logger.Logger) authn.Service {
	database := postgres.NewDatabase(db)
	repo := tracing.New(postgres.New(database), tracer)

	idp := mfidp.New()
	t := jwt.New(secret)
	svc := authn.New(repo, idp, t)
	svc = api.LoggingMiddleware(svc, logger)
	svc = api.MetricsMiddleware(
		svc,
		kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace: "authn",
			Subsystem: "api",
			Name:      "request_count",
			Help:      "Number of requests received.",
		}, []string{"method"}),
		kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
			Namespace: "authn",
			Subsystem: "api",
			Name:      "request_latency_microseconds",
			Help:      "Total duration of requests in microseconds.",
		}, []string{"method"}),
	)

	return svc
}
