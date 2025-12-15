package service

import (
	"context"
	"database/sql"
	"errors"
	"log"

	"github.com/dwikikusuma/atlas/internal/wallet/db"
	"github.com/dwikikusuma/atlas/pkg/pb/wallet"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type PostgresWalletService struct {
	wallet.UnimplementedWalletServiceServer
	pool *pgxpool.Pool
}

func NewPostgresWalletService(db *pgxpool.Pool) *PostgresWalletService {
	return &PostgresWalletService{
		pool: db,
	}
}

func (s *PostgresWalletService) execTx(ctx context.Context, fn func(*db.Queries) error) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		log.Println("failed to begin transaction:", err)
		return err
	}

	q := db.New(tx)

	if err = fn(q); err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			log.Println("failed to rollback transaction:", rbErr)
			return rbErr
		}
	}

	return tx.Commit(ctx)
}

func (s *PostgresWalletService) GetBalance(ctx context.Context, req *wallet.GetBalanceRequest) (*wallet.GetBalanceResponse, error) {
	q := db.New(s.pool)
	walletDetail, err := q.GetWallet(ctx, req.UserId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &wallet.GetBalanceResponse{
				Balance: 0,
				UserId:  req.UserId,
			}, nil
		}
		log.Println(err)
		return nil, status.Errorf(codes.Internal, "failed to get wallet: %v", err)
	}

	return &wallet.GetBalanceResponse{
		Balance: walletDetail.Balance,
	}, nil
}

func (s *PostgresWalletService) CreditBalance(ctx context.Context, req *wallet.CreditBalanceRequest) (*wallet.BalanceResponse, error) {
	var balance float64

	q := db.New(s.pool)
	err := s.execTx(ctx, func(queries *db.Queries) error {
		_, err := q.GetWallet(ctx, req.UserId)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return status.Errorf(codes.NotFound, "wallet not found: %v", err)
			}
			return status.Errorf(codes.Internal, "failed to get wallet: %v", err)
		}

		_, err = q.CreateTransaction(ctx, db.CreateTransactionParams{
			WalletID:    req.UserId,
			Amount:      req.Amount,
			Description: "CREDIT",
			ReferenceID: pgtype.Text{String: req.ReferenceId, Valid: req.ReferenceId != ""},
		})
		if err != nil {
			return status.Errorf(codes.Internal, "failed to create transaction: %v", err)
		}

		w, err := q.AddWalletBalance(ctx, db.AddWalletBalanceParams{
			UserID: req.UserId,
			Amount: req.Amount,
		})
		if err != nil {
			return status.Errorf(codes.Internal, "failed to credit balance: %v", err)
		}

		balance = w.Balance

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &wallet.BalanceResponse{
		NewBalance: balance,
	}, nil
}

func (s *PostgresWalletService) DebitBalance(ctx context.Context, req *wallet.DebitBalanceRequest) (*wallet.BalanceResponse, error) {
	var balance float64

	q := db.New(s.pool)
	err := s.execTx(ctx, func(queries *db.Queries) error {
		_, err := q.GetWallet(ctx, req.UserId)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return status.Errorf(codes.NotFound, "wallet not found: %v", err)
			}
			return status.Errorf(codes.Internal, "failed to get wallet: %v", err)
		}

		_, err = q.CreateTransaction(ctx, db.CreateTransactionParams{
			WalletID:    req.UserId,
			Amount:      -req.Amount,
			Description: "DEBIT",
			ReferenceID: pgtype.Text{String: req.ReferenceId, Valid: req.ReferenceId != ""},
		})
		if err != nil {
			return status.Errorf(codes.Internal, "failed to create transaction: %v", err)
		}

		w, err := q.AddWalletBalance(ctx, db.AddWalletBalanceParams{
			UserID: req.UserId,
			Amount: -req.Amount,
		})
		if err != nil {
			return status.Errorf(codes.Internal, "failed to debit balance: %v", err)
		}
		balance = w.Balance
		return nil
	})

	if err != nil {
		return nil, err
	}

	return &wallet.BalanceResponse{
		NewBalance: balance,
	}, nil
}
