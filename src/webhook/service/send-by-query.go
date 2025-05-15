package webhook_service

import (
	"net/http"
	"sync"

	"github.com/Astervia/wacraft-server/src/database"
	"github.com/Astervia/wacraft-core/src/repository"
	webhook_entity "github.com/Astervia/wacraft-core/src/webhook/entity"
	"gorm.io/gorm"
)

func SendAllByQuery(
	entity webhook_entity.Webhook,
	payload interface{},
) error {
	tx := database.DB.Begin()
	err := tx.Error
	if err != nil {
		return tx.Error
	}

	var whCount int64

	if err := tx.Model(&entity).Where(&entity).Count(&whCount).Error; err != nil {
		return err
	}

	offset := 0
	var offsetMu sync.Mutex
	// errCh := make(chan error, whCount)
	var wg sync.WaitGroup

	for i := 0; i < int(whCount); i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			SendByQuery(entity, payload, tx, &offset, &offsetMu)
		}()
	}

	wg.Wait()

	err = tx.Commit().Error
	if err != nil {
		tx.Rollback()
		return err
	}

	return nil
}

func SendByQuery(
	entity webhook_entity.Webhook,
	payload interface{},
	tx *gorm.DB,
	offset *int,
	offsetMu *sync.Mutex,
) error {
	var err error
	var wh webhook_entity.Webhook

	offsetMu.Lock()
	// Query webhook that satisfy the entity
	err = tx.Where(&entity).Offset(*offset).First(&wh).Error
	(*offset) = (*offset) + 1
	offsetMu.Unlock()
	if err != nil {
		return err
	}

	// Create a new HTTP client
	client := &http.Client{}

	// Execute the request and get the webhook log
	webhookLog, err := wh.ExecuteRequest(payload, client)
	if err != nil {
		return err
	}

	// Store the webhook log in the database
	_, err = repository.Create(webhookLog, tx)

	return err
}
