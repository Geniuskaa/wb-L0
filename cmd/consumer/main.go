package main

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/sync/errgroup"
	"net"
	"os"
	"os/signal"
	"syscall"
	restSrv "wb_l0/internal/server"
)

const (
	defaultPort        = "8080"
	defaultHost        = "0.0.0.0"
	defaultNatsHost    = "0.0.0.0"
	defaultNatsPort    = "4222"
	defaultPostgresDSN = "postgres://postgres:password@localhost:5432/test"
)

func main() {
	mainCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGHUP, syscall.SIGINT,
		syscall.SIGQUIT, syscall.SIGTERM)
	defer stop()

	restPort, ok := os.LookupEnv("APP_PORT")
	if !ok {
		restPort = defaultPort
	}

	restHost, ok := os.LookupEnv("APP_HOST")
	if !ok {
		restHost = defaultHost
	}

	natsHost, ok := os.LookupEnv("NATS_HOST")
	if !ok {
		natsHost = defaultNatsHost
	}

	natsPort, ok := os.LookupEnv("NATS_PORT")
	if !ok {
		natsPort = defaultNatsPort
	}

	postgresDsn, ok := os.LookupEnv("APP_POSTGRES_DSN")
	if !ok {
		postgresDsn = defaultPostgresDSN
	}

	httpSrv := applicationStart(mainCtx, net.JoinHostPort(restHost, restPort),
		fmt.Sprintf("nats://%s:%s", natsHost, natsPort), postgresDsn)

	g, gCtx := errgroup.WithContext(mainCtx)
	g.Go(func() error {
		fmt.Println("HTTP server started!")
		return httpSrv.ListenAndServe()
	})
	g.Go(func() error {
		<-gCtx.Done()
		fmt.Println("HTTP server is shut down.")
		return httpSrv.Shutdown(context.Background())
	})

	if err := g.Wait(); err != nil {
		fmt.Printf("exit reason: %s \n", err)
	}
	fmt.Println("Servers were gracefully shut down.")
}

func applicationStart(ctx context.Context, addr string, natsAddr string, dsn string) *restSrv.Server {

	nc, err := nats.Connect(natsAddr)
	if err != nil {
		panic(fmt.Errorf("Error establishing connection to NATS streaming: %w", err))
	}

	logger := loggerInit()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		panic(fmt.Errorf("Error establishing connection to Postgres: %w", err))
	}

	restServer := restSrv.NewServer(ctx, addr, logger, nc, pool)

	return restServer
}

func loggerInit() *zap.SugaredLogger {
	encoderConfig := zap.NewDevelopmentEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder

	consoleEncoder := zapcore.NewConsoleEncoder(encoderConfig)
	core := zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), zapcore.ErrorLevel)

	sugarLogger := zap.New(core).Sugar()

	return sugarLogger
}
