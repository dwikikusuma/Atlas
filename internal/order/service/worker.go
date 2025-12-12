package service

import (
	"context"
	"encoding/json"
	"log"

	"github.com/dwikikusuma/atlas/internal/order/db"
	"github.com/dwikikusuma/atlas/pkg/kafka"
	"github.com/jackc/pgx/v5/pgtype"

	dispatchModel "github.com/dwikikusuma/atlas/internal/dispatch/model"
)

type OrderWorker struct {
	consumer kafka.EventConsumer
	store    db.Querier
}

func NewOrderWorker(consumer kafka.EventConsumer, store db.Querier) *OrderWorker {
	return &OrderWorker{
		consumer: consumer,
		store:    store,
	}
}

func (o *OrderWorker) Start(ctx context.Context) error {
	log.Println("Starting order worker...")
	for {
		select {
		case <-ctx.Done():
			log.Println("Order worker stopping...")
			return nil
		default:
		}

		var model dispatchModel.RideDispatchedEvent

		m, err := o.consumer.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			log.Print("Error fetching message")
			continue
		}

		err = json.Unmarshal(m.Value, &model)
		if err != nil {
			log.Printf("❌ Failed to parse JSON for key=%s: %v", string(m.Key), err)
			if err = o.consumer.CommitMessages(ctx, m); err != nil {
				log.Printf("Failed to commit messages for key=%s: %v", string(m.Key), err)
			}
			continue
		}

		var uuidOrder pgtype.UUID
		if err = uuidOrder.Scan(model.RideID); err != nil {
			log.Printf("❌ Invalid RideID format for key=%s: %v", string(m.Key), err)
			if err = o.consumer.CommitMessages(ctx, m); err != nil {
				log.Printf("Failed to commit messages for key=%s: %v", string(m.Key), err)
			}
			continue
		}

		args := db.UpdateOrderDriverParams{
			ID:       uuidOrder,
			DriverID: pgtype.Text{String: model.DriverID, Valid: true},
		}

		if err = o.store.UpdateOrderDriver(ctx, args); err != nil {
			log.Printf("❌ Failed to update order for RideID=%s: %v", model.RideID, err)
			continue
		}

		log.Printf("✅ Processed RideDispatchedEvent for RideID=%s, DriverID=%s", model.RideID, model.DriverID)

		if err = o.consumer.CommitMessages(ctx, m); err != nil {
			log.Printf("Failed to commit messages for key=%s: %v", string(m.Key), err)
		}

	}
}
