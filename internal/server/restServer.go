package server

import (
	"context"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
	"net/http"
	"wb_l0/internal/db"
	"wb_l0/internal/order"
)

type Server struct {
	ctx    context.Context
	logger *zap.SugaredLogger
	mux    *chi.Mux
	*http.Server
}

func (s *Server) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	s.mux.ServeHTTP(writer, request)
}

func NewServer(ctx context.Context, addr string, logger *zap.SugaredLogger, nc *nats.Conn, pool *pgxpool.Pool) *Server {
	psql := db.NewDB(ctx, pool)

	service := order.NewService(ctx, psql, nc, logger)
	service.Init()

	mux := chi.NewRouter()

	httpSrv := http.Server{
		Addr:    addr,
		Handler: mux,
	}

	serv := Server{
		ctx:    ctx,
		logger: logger,
		mux:    mux,
		Server: &httpSrv,
	}

	mux.With(serv.recoverer).Mount("/api/order", order.NewHandler(ctx, logger, service).Routes())
	return &serv
}

func (s *Server) recoverer(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {

		defer func() {
			if err := recover(); err != nil {
				writer.WriteHeader(http.StatusInternalServerError)
				writer.Write([]byte("Something going wrong..."))
				s.logger.Error("panic occurred:", err)
			}
		}()
		handler.ServeHTTP(writer, request)
	})
}
