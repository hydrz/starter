package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const rollbackTimeout = 3 * time.Second

// Transactor 定义在原子事务中执行业务逻辑的抽象接口。
// 业务/应用服务层只依赖此接口，不依赖底层具体的数据库或事务对象。
type Transactor interface {
	WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}

type txContextKey struct{}

// PgxTransactor 基于 pgxpool.Pool 实现 Transactor。
type PgxTransactor struct {
	pool *pgxpool.Pool
}

// NewTransactor 返回一个新的 PgxTransactor 实例。
func NewTransactor(pool *pgxpool.Pool) *PgxTransactor {
	return &PgxTransactor{pool: pool}
}

// WithinTransaction 在事务上下文中执行给定的闭包函数。
// 如果上下文已经包含活跃事务，则直接复用，避免重复开启；
// 发生 panic 或返回 error 时自动回滚，正常结束时自动提交。
func (t *PgxTransactor) WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) (err error) {
	if _, ok := TxFromContext(ctx); ok {
		return fn(ctx)
	}

	tx, err := t.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			rollbackCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), rollbackTimeout)
			defer cancel()
			_ = tx.Rollback(rollbackCtx)
			panic(p)
		} else if err != nil {
			rollbackCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), rollbackTimeout)
			defer cancel()
			_ = tx.Rollback(rollbackCtx)
		}
	}()

	txCtx := context.WithValue(ctx, txContextKey{}, tx)
	if err = fn(txCtx); err != nil {
		return err
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

// TxFromContext 从 context.Context 中提取活跃的 pgx.Tx（如果存在）。
func TxFromContext(ctx context.Context) (pgx.Tx, bool) {
	tx, ok := ctx.Value(txContextKey{}).(pgx.Tx)
	return tx, ok
}
