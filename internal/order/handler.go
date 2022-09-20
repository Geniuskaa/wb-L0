package order

import (
	"context"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
	"html/template"
	"net/http"
)

type handler struct {
	ctx     context.Context
	log     *zap.SugaredLogger
	service *Service
}

func NewHandler(ctx context.Context, log *zap.SugaredLogger, service *Service) *handler {
	return &handler{ctx: ctx, log: log, service: service}
}

func (h *handler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Get("/{id}", h.GetOrderById)

	return r
}

func (h *handler) GetOrderById(writer http.ResponseWriter, request *http.Request) {
	orderId := chi.URLParam(request, "id")
	order := h.service.GetOrder(orderId)
	if order == nil {
		http.Error(writer, "Error getting order", http.StatusInternalServerError)
		return
	}

	ts, err := template.ParseFiles("./ui/html/index.html")
	if err != nil {
		http.Error(writer, "Error preparing html", http.StatusInternalServerError)
		return
	}

	err = ts.Execute(writer, order)
	if err != nil {
		http.Error(writer, "Error preparing html", http.StatusInternalServerError)
		return
	}

	return
}
