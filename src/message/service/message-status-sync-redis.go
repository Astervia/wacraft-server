package message_service

import (
	"context"
	"errors"
	"fmt"
	"time"

	synch_redis "github.com/Astervia/wacraft-core/src/synch/redis"
	message_model "github.com/Rfluid/whatsapp-cloud-api/src/message"
	"github.com/google/uuid"
)

// RedisMessageStatusSync implements MessageStatusSync using Redis lists
// as single-use channels (BLPOP/RPUSH rendezvous pattern).
//
// For each wamID the following Redis keys are used:
//
//	{prefix}msg:{wamID}:status  – written by AddStatus, read by AddMessage
//	{prefix}msg:{wamID}:saved   – written by MessageSaved/RollbackMessage, read by AddStatus
//
// After the handshake both keys are deleted. Key TTL (60 s) acts as safety
// garbage-collection for abandoned flows.
type RedisMessageStatusSync struct {
	client *synch_redis.Client
}

const redisSyncKeyTTL = 60 * time.Second

func (r *RedisMessageStatusSync) statusKey(wamID string) string {
	return r.client.PrefixKey("msg:" + wamID + ":status")
}

func (r *RedisMessageStatusSync) savedKey(wamID string) string {
	return r.client.PrefixKey("msg:" + wamID + ":saved")
}

// AddMessage blocks until AddStatus signals arrival (BLPOP on the status key).
func (r *RedisMessageStatusSync) AddMessage(wamID string, timeout time.Duration) error {
	ctx := context.Background()
	rdb := r.client.Redis()

	// Set a safety TTL on the status key so it is cleaned up even if
	// AddStatus never arrives (e.g. dropped webhook).
	_ = rdb.Expire(ctx, r.statusKey(wamID), redisSyncKeyTTL)

	result, err := rdb.BLPop(ctx, timeout, r.statusKey(wamID)).Result()
	if err != nil {
		return fmt.Errorf(
			"timeout waiting for whatsapp message status update. Waited %s for WhatsApp to update the message status and did not receive any updates: %w",
			timeout.String(), err,
		)
	}
	_ = result // value is the sentinel empty string pushed by AddStatus
	return nil
}

// MessageSaved signals that the message was persisted. Pushes the message ID
// onto the saved key so AddStatus can unblock.
func (r *RedisMessageStatusSync) MessageSaved(wamID string, messageID uuid.UUID, timeout time.Duration) error {
	ctx := context.Background()
	rdb := r.client.Redis()

	savedKey := r.savedKey(wamID)
	count, err := rdb.RPush(ctx, savedKey, messageID.String()).Result()
	if err != nil || count == 0 {
		return fmt.Errorf(
			"timeout waiting to signal message saved. Signaling that the message was saved took %s: %w",
			timeout.String(), err,
		)
	}

	// Set a safety TTL and clean up the status key.
	_ = rdb.Expire(ctx, savedKey, redisSyncKeyTTL)
	_ = rdb.Del(ctx, r.statusKey(wamID))
	return nil
}

// RollbackMessage signals that the message was NOT persisted by pushing an
// empty string onto the saved key. AddStatus will return an error.
func (r *RedisMessageStatusSync) RollbackMessage(wamID string, timeout time.Duration) error {
	ctx := context.Background()
	rdb := r.client.Redis()

	savedKey := r.savedKey(wamID)
	count, err := rdb.RPush(ctx, savedKey, "").Result()
	if err != nil || count == 0 {
		return fmt.Errorf(
			"timeout waiting to signal message rolledback. Signaling took %s: %w",
			timeout.String(), err,
		)
	}

	_ = rdb.Expire(ctx, savedKey, redisSyncKeyTTL)
	_ = rdb.Del(ctx, r.statusKey(wamID))
	return nil
}

// AddStatus signals that a status update arrived (RPUSH to status key) then
// blocks waiting for the message ID (BLPOP on the saved key).
func (r *RedisMessageStatusSync) AddStatus(wamID string, status *message_model.SendingStatus, timeout time.Duration) (uuid.UUID, error) {
	if status == nil {
		return uuid.Nil, errors.New("status is nil")
	}

	ctx := context.Background()
	rdb := r.client.Redis()

	// Signal that the status arrived.
	statusKey := r.statusKey(wamID)
	if err := rdb.RPush(ctx, statusKey, "").Err(); err != nil {
		return uuid.Nil, fmt.Errorf(
			"timeout waiting to signal status added. Signaling took %s: %w",
			timeout.String(), err,
		)
	}
	_ = rdb.Expire(ctx, statusKey, redisSyncKeyTTL)

	// Block until MessageSaved or RollbackMessage.
	savedKey := r.savedKey(wamID)
	result, err := rdb.BLPop(ctx, timeout, savedKey).Result()
	if err != nil {
		return uuid.Nil, fmt.Errorf(
			"timeout waiting for message to be added. Waiting took %s: %w",
			timeout.String(), err,
		)
	}

	// result[0] = key name, result[1] = value
	messageIDStr := result[1]
	_ = rdb.Del(ctx, savedKey) // cleanup

	if messageIDStr == "" {
		return uuid.Nil, errors.New("message rolled back")
	}

	messageID, err := uuid.Parse(messageIDStr)
	if err != nil {
		return uuid.Nil, err
	}

	return messageID, nil
}

// NewRedisMessageStatusSync creates a Redis-backed MessageStatusSync.
func NewRedisMessageStatusSync(client *synch_redis.Client) MessageStatusSync {
	return &RedisMessageStatusSync{client: client}
}
