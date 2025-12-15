package service

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/dwikikusuma/atlas/internal/wallet/model"
	"github.com/dwikikusuma/atlas/pkg/kafka"
	"github.com/dwikikusuma/atlas/pkg/pb/wallet"
)

type WalletWorker struct {
	consumer *kafka.Consumer
	service  WalletService
}

func NewWalletWorker(consumer *kafka.Consumer, service WalletService) *WalletWorker {
	return &WalletWorker{
		consumer: consumer,
		service:  service,
	}
}

func (w *WalletWorker) Start(ctx context.Context) error {
	defer w.consumer.Close()
	log.Println("Starting wallet worker...")

	for {
		select {
		case <-ctx.Done():
			log.Println("Wallet worker stopping...")
			return nil
		default:
		}

		var event model.DebitBalanceEvent

		m, err := w.consumer.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			log.Printf("❌ Error fetching message: %v", err)
			continue
		}

		if err = json.Unmarshal(m.Value, &event); err != nil {
			log.Printf("❌ Failed to parse JSON for key=%s: %v", string(m.Key), err)
			if err = w.consumer.CommitMessages(ctx, m); err != nil {
				log.Printf("⚠️ Failed to commit malformed message for key=%s: %v", string(m.Key), err)
			}
			continue
		}

		if event.UserID == "" || event.Amount <= 0 || event.Reference == "" {
			log.Printf("❌ Invalid event data: userID=%s, amount=%.2f, ref=%s",
				event.UserID, event.Amount, event.Reference)
			if err = w.consumer.CommitMessages(ctx, m); err != nil {
				log.Printf("⚠️ Failed to commit invalid message for key=%s: %v", string(m.Key), err)
			}
			continue
		}

		debitCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		args := wallet.DebitBalanceRequest{
			UserId:      event.UserID,
			Amount:      event.Amount,
			ReferenceId: event.Reference,
		}

		_, err = w.service.DebitBalance(debitCtx, &args)
		cancel()

		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			log.Printf("❌ Failed to debit balance for userID=%s, ref=%s: %v",
				event.UserID, event.Reference, err)
			continue
		}

		log.Printf("✅ Processed debit: userID=%s, amount=%.2f, ref=%s",
			event.UserID, event.Amount, event.Reference)

		if err = w.consumer.CommitMessages(ctx, m); err != nil {
			log.Printf("⚠️ Failed to commit message for key=%s: %v", string(m.Key), err)
		}
	}
}
