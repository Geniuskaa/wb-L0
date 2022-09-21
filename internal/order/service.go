package order

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
	"strings"
	"wb_l0/internal/db"
	"wb_l0/internal/dto"
)

type Service struct {
	ctx    context.Context
	repo   db.Repo
	nc     *nats.Conn
	logger *zap.SugaredLogger
	cache  map[string]*dto.OrderDTO
}

func NewService(ctx context.Context, repo db.Repo, nc *nats.Conn, logger *zap.SugaredLogger) *Service {
	return &Service{ctx: ctx, repo: repo, nc: nc, logger: logger, cache: make(map[string]*dto.OrderDTO, db.MaxCountOfOrders)}
}

func (s *Service) Init() {
	s.CacheWarmUp()
	s.getOrder()
}

func (s *Service) CacheWarmUp() {
	orders, err := s.repo.GetAllOrders()
	if err != nil {
		s.logger.Error(fmt.Errorf("CacheWarmUp failed: %w", err))
		return
	}

	for _, val := range orders {
		s.cache[val.OrderUID] = val
	}
}

func (s *Service) getOrder() {
	sub, err := s.nc.SubscribeSync("orders")
	if err != nil {
		s.logger.Error(fmt.Errorf("GetOrder failed: %w", err))
		return
	}

	go func() {
		fmt.Println("Consumer started working!")
		for {
			var msg *nats.Msg
			select {
			case <-s.ctx.Done():
				fmt.Println("Consumer finished his work!")
				return
			default:
				msg, err = sub.NextMsgWithContext(s.ctx)
				if err != nil {
					s.logger.Error(fmt.Errorf("GetOrder failed during msg recieve: %w", err))
					continue
				}
			}

			order := dto.OrderDTO{}
			err := json.Unmarshal(msg.Data, &order)
			if err != nil {
				s.logger.Error(fmt.Errorf("GetOrder failed during msg unmarshaling: %w", err))
				continue
			}

			valid := orderValidate(&order)
			// Получили неверные данные, залогировали, продолжили работать
			if !valid {
				s.logger.Warn("GetOrder recieved uncorrect data from NATS stream")
				continue
			}

			err = s.repo.UploadOrder(&order)
			if err != nil {
				s.logger.Error(fmt.Errorf("GetOrder failed during uploading order to DB: %w", err))
				continue
			}

			// Если выше возникли проблемы с добавлением в БД, значит и в кэш значение не должно попасть
			s.cache[order.OrderUID] = &order
		}

	}()
}

func (s *Service) GetOrder(orderUID string) *dto.OrderDTO {
	if v, exist := s.cache[orderUID]; exist {
		return v
	} else {
		order, err := s.repo.GetOrder(orderUID)
		if err != nil {
			s.logger.Error(fmt.Errorf("GetOrder failed: %w", err))
			return nil
		}
		return order
	}
}

func orderValidate(order *dto.OrderDTO) bool {
	if strings.EqualFold(order.OrderUID, "") {
		return false
	} else if strings.EqualFold(order.TrackNumber, "") {
		return false
	} else if strings.EqualFold(order.Delivery.Phone, "") {
		return false
	} else if strings.EqualFold(order.Payment.Transaction, "") {
		return false
	} else if len(order.Items) == 0 {
		return false
	} else if order.DateCreated.Year() == 1 {
		return false
	}

	order.Delivery.Phone = strings.TrimLeft(order.Delivery.Phone, "+")
	return true
}
