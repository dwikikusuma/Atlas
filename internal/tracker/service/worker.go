package service

import (
	"context"
	"encoding/json"
	"log"

	"github.com/dwikikusuma/atlas/internal/tracker/domain"
	"github.com/dwikikusuma/atlas/internal/tracker/model"
	"github.com/dwikikusuma/atlas/pkg/kafka"
)

type IngestionWorker struct {
	consumer kafka.EventConsumer
	repo     domain.LocationRepository
}

func NewIngestionWorker(consumer kafka.EventConsumer, repo domain.LocationRepository) *IngestionWorker {
	return &IngestionWorker{
		consumer: consumer,
		repo:     repo,
	}
}

func (w *IngestionWorker) Run(ctx context.Context) {
	for {
		msg, err := w.consumer.FetchMessage(ctx)
		if err != nil {
			log.Printf("Error fetching message: %v", err)
			continue
		}

		var event model.LocationEvent
		err = json.Unmarshal(msg.Value, &event)
		if err != nil {
			log.Printf("Error unmarshaling message: %v", err)
			continue
		}

		err = w.repo.UpdatePosition(ctx, event.UserID, event.Latitude, event.Longitude)
		if err != nil {
			log.Printf("Error updating position: %v", err)
			continue
		}

		err = w.consumer.CommitMessages(ctx, msg)
		if err != nil {
			log.Printf("Error committing message: %v", err)
		}
	}
}
