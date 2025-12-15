package service

import (
	"context"

	"github.com/dwikikusuma/atlas/pkg/pb/wallet"
)

type WalletService interface {
	GetBalance(ctx context.Context, req *wallet.GetBalanceRequest) (*wallet.GetBalanceResponse, error)
	CreditBalance(ctx context.Context, req *wallet.CreditBalanceRequest) (*wallet.BalanceResponse, error)
	DebitBalance(ctx context.Context, req *wallet.DebitBalanceRequest) (*wallet.BalanceResponse, error)
}
