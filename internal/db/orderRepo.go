package db

import (
	"fmt"
	"github.com/jackc/pgx/v5"
	"wb_l0/internal/dto"
)

const maxCountOfOrders = 100

type OrderRepo interface {
	UploadOrder(ord *dto.OrderDTO) error
	GetAllOrders() ([]*dto.OrderDTO, error)
	GetOrder(orderUID string) (*dto.OrderDTO, error)
}

func (d *DB) UploadOrder(ord *dto.OrderDTO) error {
	tx, err := d.Pool.BeginTx(d.ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead})
	if err != nil {
		errOfRollback := tx.Rollback(d.ctx)
		if errOfRollback != nil {
			return fmt.Errorf("UploadOrder failed: %w", errOfRollback)
		}
		return fmt.Errorf("UploadOrder failed: %w", err)
	}

	row := tx.QueryRow(d.ctx, `INSERT INTO payments (transaction, request_id, currency, provider, amount, payment_dt, 
            bank, delivery_cost, goods_total, custom_fee) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) RETURNING id`,
		ord.Payment.Transaction, ord.Payment.RequestID, ord.Payment.Currency, ord.Payment.Provider, ord.Payment.Amount,
		ord.Payment.PaymentDt, ord.Payment.Bank, ord.Payment.DeliveryCost, ord.Payment.GoodsTotal, ord.Payment.CustomFee)

	var paymentId int64
	err = row.Scan(&paymentId)
	if err != nil {
		errOfRollback := tx.Rollback(d.ctx)
		if errOfRollback != nil {
			return fmt.Errorf("UploadOrder failed: %w", errOfRollback)
		}
		return fmt.Errorf("UploadOrder failed: %w", err)
	}

	row = tx.QueryRow(d.ctx, `INSERT INTO clients (full_name, phone, zip, city, address, region, email) VALUES 
            ($1, $2, $3, $4, $5, $6, $7) RETURNING id`, ord.Delivery.Name, ord.Delivery.Phone, ord.Delivery.Zip,
		ord.Delivery.City, ord.Delivery.Address, ord.Delivery.Region, ord.Delivery.Email)

	var clientId int64
	err = row.Scan(&clientId)
	if err != nil {
		errOfRollback := tx.Rollback(d.ctx)
		if errOfRollback != nil {
			return fmt.Errorf("UploadOrder failed: %w", errOfRollback)
		}
		return fmt.Errorf("UploadOrder failed: %w", err)
	}

	row = tx.QueryRow(d.ctx, `INSERT INTO orders (order_uid, track_number, entry, delivery, payment, locale, 
            internal_signature, customer_id, delivery_service, shardkey, sm_id, date_created, oof_shard) VALUES 
            ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13) RETURNING id`, ord.OrderUID, ord.TrackNumber,
		ord.Entry, clientId, paymentId, ord.Locale, ord.InternalSignature, ord.CustomerID, ord.DeliveryService,
		ord.Shardkey, ord.SmID, ord.DateCreated, ord.OofShard)

	var orderId int64
	err = row.Scan(&orderId)
	if err != nil {
		errOfRollback := tx.Rollback(d.ctx)
		if errOfRollback != nil {
			return fmt.Errorf("UploadOrder failed: %w", errOfRollback)
		}
		return fmt.Errorf("UploadOrder failed: %w", err)
	}

	for _, val := range ord.Items {
		row := tx.QueryRow(d.ctx, `INSERT INTO items (track_number, price, rid, name, sale, size, total_price, 
                nm_id, brand, status) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) RETURNING id`, val.TrackNumber,
			val.Price, val.Rid, val.Name, val.Sale, val.Size, val.TotalPrice, val.NmID, val.Brand, val.Status)

		var itemId int64
		err = row.Scan(&itemId)
		if err != nil {
			errOfRollback := tx.Rollback(d.ctx)
			if errOfRollback != nil {
				return fmt.Errorf("UploadOrder failed: %w", errOfRollback)
			}
			return fmt.Errorf("UploadOrder failed: %w", err)
		}

		_, err := tx.Exec(d.ctx, `INSERT INTO order_to_items (order_id, item_id) VALUES ($1, $2)`, orderId, itemId)
		if err != nil {
			errOfRollback := tx.Rollback(d.ctx)
			if errOfRollback != nil {
				return fmt.Errorf("UploadOrder failed: %w", errOfRollback)
			}
			return fmt.Errorf("UploadOrder failed: %w", err)
		}
	}

	err = tx.Commit(d.ctx)
	if err != nil {
		errOfRollback := tx.Rollback(d.ctx)
		if errOfRollback != nil {
			return fmt.Errorf("UploadOrder failed: %w", errOfRollback)
		}
		return fmt.Errorf("UploadOrder failed: %w", err)
	}

	return nil
}

func (d *DB) GetAllOrders() ([]*dto.OrderDTO, error) {
	orders := make([]*dto.OrderDTO, 0, maxCountOfOrders)
	rows, err := d.Query(d.ctx, `SELECT id, order_uid, track_number, entry, delivery, payment, locale, internal_signature, 
       customer_id, delivery_service,shardkey, sm_id, date_created, oof_shard from orders ORDER BY date_created DESC LIMIT 100`)
	defer rows.Close()

	if err != nil {
		return nil, fmt.Errorf("GetAllOrders failed during getting orders: %w", err)
	}

	for rows.Next() {
		deliveryId := 0
		paymentId := 0
		orderId := 0

		order := dto.OrderDTO{}

		err = rows.Scan(&orderId, &order.OrderUID, &order.TrackNumber, &order.Entry, &deliveryId, &paymentId,
			&order.Locale, &order.InternalSignature, &order.CustomerID, &order.DeliveryService, &order.Shardkey,
			&order.SmID, &order.DateCreated, &order.OofShard)
		if err != nil {
			return nil, fmt.Errorf("GetAllOrders failed: %w", err)
		}

		deliv := &order.Delivery
		row := d.QueryRow(d.ctx, `SELECT full_name, phone, zip, city, address, region, email FROM clients WHERE id=$1`,
			deliveryId)
		err = row.Scan(&deliv.Name, &deliv.Phone, &deliv.Zip, &deliv.City, &deliv.Address, &deliv.Region, &deliv.Email)
		if err != nil {
			return nil, fmt.Errorf("GetAllOrders failed: %w", err)
		}

		paym := &order.Payment
		row = d.QueryRow(d.ctx, `SELECT transaction, request_id, currency, provider, amount, payment_dt, bank, 
       		delivery_cost, goods_total, custom_fee FROM payments WHERE id=$1`, paymentId)
		err = row.Scan(&paym.Transaction, &paym.RequestID, &paym.Currency, &paym.Provider, &paym.Amount, &paym.PaymentDt,
			&paym.Bank, &paym.DeliveryCost, &paym.GoodsTotal, &paym.CustomFee)

		rowsItemsIds, err := d.Query(d.ctx, `SELECT item_id FROM order_to_items WHERE order_id=$1`, orderId)
		if err != nil {
			return nil, fmt.Errorf("GetAllOrders failed: %w", err)
		}

		for rowsItemsIds.Next() {
			id := 0
			err = rowsItemsIds.Scan(&id)
			if err != nil {
				return nil, fmt.Errorf("GetAllOrders failed: %w", err)
			}

			row = d.QueryRow(d.ctx, `SELECT track_number, price, rid, name, sale, size, total_price, nm_id, brand, 
       			status FROM items WHERE id=$1`, id)

			item := dto.ItemDTO{}

			err = row.Scan(&item.TrackNumber, &item.Price, &item.Rid, &item.Name, &item.Sale, &item.Size, &item.TotalPrice,
				&item.NmID, &item.Brand, &item.Status)
			if err != nil {
				return nil, fmt.Errorf("GetAllOrders failed: %w", err)
			}

			order.Items = append(order.Items, item)
		}

		orders = append(orders, &order)
	}

	return orders, nil
}

func (d *DB) GetOrder(orderUID string) (*dto.OrderDTO, error) {
	row := d.QueryRow(d.ctx, `SELECT * from orders WHERE order_uid=$1`, orderUID)

	deliveryId := 0
	paymentId := 0
	orderId := 0

	order := dto.OrderDTO{}

	err := row.Scan(&orderId, &order.OrderUID, &order.TrackNumber, &order.Entry, &deliveryId, &paymentId,
		&order.Locale, &order.InternalSignature, &order.CustomerID, &order.DeliveryService, &order.Shardkey,
		&order.SmID, &order.DateCreated, &order.OofShard)
	if err != nil {
		return nil, fmt.Errorf("GetOrder failed: %w", err)
	}

	deliv := &order.Delivery
	row = d.QueryRow(d.ctx, `SELECT full_name, phone, zip, city, address, region, email FROM clients WHERE id=$1`,
		deliveryId)
	err = row.Scan(&deliv.Name, &deliv.Phone, &deliv.Zip, &deliv.City, &deliv.Address, &deliv.Region, &deliv.Email)
	if err != nil {
		return nil, fmt.Errorf("GetAllOrders failed: %w", err)
	}

	paym := &order.Payment
	row = d.QueryRow(d.ctx, `SELECT transaction, request_id, currency, provider, amount, payment_dt, bank, 
       		delivery_cost, goods_total, custom_fee FROM payments WHERE id=$1`, paymentId)
	err = row.Scan(&paym.Transaction, &paym.RequestID, &paym.Currency, &paym.Provider, &paym.Amount, &paym.PaymentDt,
		&paym.Bank, &paym.DeliveryCost, &paym.GoodsTotal, &paym.CustomFee)

	rowsItemsIds, err := d.Query(d.ctx, `SELECT item_id FROM order_to_items WHERE order_id=$1`, orderId)
	if err != nil {
		return nil, fmt.Errorf("GetAllOrders failed: %w", err)
	}

	for rowsItemsIds.Next() {
		id := 0
		err = rowsItemsIds.Scan(&id)
		if err != nil {
			return nil, fmt.Errorf("GetAllOrders failed: %w", err)
		}

		row = d.QueryRow(d.ctx, `SELECT track_number, price, rid, name, sale, size, total_price, nm_id, brand, 
       			status FROM items WHERE id=$1`, id)

		item := dto.ItemDTO{}

		err = row.Scan(&item.TrackNumber, &item.Price, &item.Rid, &item.Name, &item.Sale, &item.Size, &item.TotalPrice,
			&item.NmID, &item.Brand, &item.Status)
		if err != nil {
			return nil, fmt.Errorf("GetAllOrders failed: %w", err)
		}

		order.Items = append(order.Items, item)
	}

	return &order, nil
}
