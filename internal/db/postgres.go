package db

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DB struct {
	ctx context.Context
	*pgxpool.Pool
	Repo
}

type Repo interface {
	OrderRepo
}

func NewDB(ctx context.Context, pool *pgxpool.Pool) *DB {
	return &DB{ctx: ctx, Pool: pool}
}
