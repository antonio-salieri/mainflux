// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"time"

	gocoap "github.com/dustin/go-coap"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/coap"
	"github.com/mainflux/mainflux/coap/api"
	"github.com/mainflux/mainflux/coap/nats"
	"github.com/mainflux/mainflux/internal/pkg/server"
	logger "github.com/mainflux/mainflux/logger"
	thingsapi "github.com/mainflux/mainflux/things/api/auth/grpc"
	opentracing "github.com/opentracing/opentracing-go"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	jconfig "github.com/uber/jaeger-client-go/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	broker "github.com/nats-io/go-nats"
)

const (
	defPort          = "5683"
	defNatsURL       = broker.DefaultURL
	defThingsURL     = "localhost:8181"
	defLogLevel      = "error"
	defClientTLS     = "false"
	defCACerts       = ""
	defPingPeriod    = "12"
	defJaegerURL     = ""
	defThingsTimeout = "1" // in seconds

	envPort          = "MF_COAP_ADAPTER_PORT"
	envNatsURL       = "MF_NATS_URL"
	envThingsURL     = "MF_THINGS_URL"
	envLogLevel      = "MF_COAP_ADAPTER_LOG_LEVEL"
	envClientTLS     = "MF_COAP_ADAPTER_CLIENT_TLS"
	envCACerts       = "MF_COAP_ADAPTER_CA_CERTS"
	envPingPeriod    = "MF_COAP_ADAPTER_PING_PERIOD"
	envJaegerURL     = "MF_JAEGER_URL"
	envThingsTimeout = "MF_COAP_ADAPTER_THINGS_TIMEOUT"
)

type config struct {
	port          string
	natsURL       string
	thingsURL     string
	logLevel      string
	clientTLS     bool
	caCerts       string
	pingPeriod    time.Duration
	jaegerURL     string
	thingsTimeout time.Duration
}

func main() {
	cfg := loadConfig()

	logger, err := logger.New(os.Stdout, cfg.logLevel)
	if err != nil {
		log.Fatalf(err.Error())
	}

	nc, err := broker.Connect(cfg.natsURL)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to NATS: %s", err))
		os.Exit(1)
	}
	defer nc.Close()

	conn := connectToThings(cfg, logger)
	defer conn.Close()

	thingsTracer, thingsCloser := initJaeger("things", cfg.jaegerURL, logger)
	defer thingsCloser.Close()

	cc := thingsapi.NewClient(conn, thingsTracer, cfg.thingsTimeout)
	respChan := make(chan string, 10000)
	pubsub := nats.New(nc)
	svc := coap.New(pubsub, cc, respChan)
	svc = api.LoggingMiddleware(svc, logger)

	svc = api.MetricsMiddleware(
		svc,
		kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace: "coap_adapter",
			Subsystem: "api",
			Name:      "request_count",
			Help:      "Number of requests received.",
		}, []string{"method"}),
		kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
			Namespace: "coap_adapter",
			Subsystem: "api",
			Name:      "request_latency_microseconds",
			Help:      "Total duration of requests in microseconds.",
		}, []string{"method"}),
	)

	errs := make(chan error, 2)

	httpServer := server.NewHTTPServer(fmt.Sprintf(":%s", cfg.port), api.MakeHTTPHandler(), "", "")
	go httpServer.Start(logger, errs)

	// TODO: Make COAP server gracefully stoppable
	go startCOAPServer(cfg, svc, cc, respChan, logger, errs)

	server.Monitor(logger, errs, httpServer)

	logger.Info("CoAP adapter terminated")
}

func loadConfig() config {
	tls, err := strconv.ParseBool(mainflux.Env(envClientTLS, defClientTLS))
	if err != nil {
		log.Fatalf("Invalid value passed for %s\n", envClientTLS)
	}

	pp, err := strconv.ParseInt(mainflux.Env(envPingPeriod, defPingPeriod), 10, 64)
	if err != nil {
		log.Fatalf("Invalid value passed for %s\n", envPingPeriod)
	}

	if pp < 1 || pp > 24 {
		log.Fatalf("Value of %s must be between 1 and 24", envPingPeriod)
	}

	timeout, err := strconv.ParseInt(mainflux.Env(envThingsTimeout, defThingsTimeout), 10, 64)
	if err != nil {
		log.Fatalf("Invalid %s value: %s", envThingsTimeout, err.Error())
	}

	return config{
		thingsURL:     mainflux.Env(envThingsURL, defThingsURL),
		natsURL:       mainflux.Env(envNatsURL, defNatsURL),
		port:          mainflux.Env(envPort, defPort),
		logLevel:      mainflux.Env(envLogLevel, defLogLevel),
		clientTLS:     tls,
		caCerts:       mainflux.Env(envCACerts, defCACerts),
		pingPeriod:    time.Duration(pp),
		jaegerURL:     mainflux.Env(envJaegerURL, defJaegerURL),
		thingsTimeout: time.Duration(timeout) * time.Second,
	}
}

func connectToThings(cfg config, logger logger.Logger) *grpc.ClientConn {
	var opts []grpc.DialOption
	if cfg.clientTLS {
		if cfg.caCerts != "" {
			tpc, err := credentials.NewClientTLSFromFile(cfg.caCerts, "")
			if err != nil {
				logger.Error(fmt.Sprintf("Failed to load certs: %s", err))
				os.Exit(1)
			}
			opts = append(opts, grpc.WithTransportCredentials(tpc))
		}
	} else {
		logger.Info("gRPC communication is not encrypted")
		opts = append(opts, grpc.WithInsecure())
	}

	conn, err := grpc.Dial(cfg.thingsURL, opts...)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to things service: %s", err))
		os.Exit(1)
	}
	return conn
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
		logger.Error(fmt.Sprintf("Failed to init Jaeger client: %s", err))
		os.Exit(1)
	}

	return tracer, closer
}

func startCOAPServer(cfg config, svc coap.Service, auth mainflux.ThingsServiceClient, respChan chan<- string, l logger.Logger, errs chan error) {
	p := fmt.Sprintf(":%s", cfg.port)
	l.Info(fmt.Sprintf("CoAP adapter service started, exposed port %s", cfg.port))
	errs <- gocoap.ListenAndServe("udp", p, api.MakeCOAPHandler(svc, auth, l, respChan, cfg.pingPeriod))
}
