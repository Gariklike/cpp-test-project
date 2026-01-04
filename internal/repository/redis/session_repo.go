package redis

import (
	"authorization-server/internal/models"
	"context"
	"encoding/json"
	"time"

	"github.com/go-redis/redis/v8"
)

type SessionRepository struct {
	client *redis.Client
	ctx    context.Context
}

func NewSessionRepository(redisURL string) (*SessionRepository, error) {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}

	client := redis.NewClient(opt)
	ctx := context.Background()

	// Проверяем подключение
	_, err = client.Ping(ctx).Result()
	if err != nil {
		return nil, err
	}

	return &SessionRepository{
		client: client,
		ctx:    ctx,
	}, nil
}

func (r *SessionRepository) Close() error {
	return r.client.Close()
}

func (r *SessionRepository) SaveAuthSession(state string, session models.AuthSession) error {
	data, err := json.Marshal(session)
	if err != nil {
		return err
	}

	return r.client.Set(r.ctx, "auth_session:"+state, data, 10*time.Minute).Err()
}

func (r *SessionRepository) GetAuthSession(state string) (*models.AuthSession, error) {
	data, err := r.client.Get(r.ctx, "auth_session:"+state).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}

	var session models.AuthSession
	err = json.Unmarshal([]byte(data), &session)
	return &session, err
}

func (r *SessionRepository) DeleteAuthSession(state string) error {
	return r.client.Del(r.ctx, "auth_session:"+state).Err()
}
