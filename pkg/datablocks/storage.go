package datablocks

import (
	"context"
	"fmt"
	"sync"
)

type KeyValStorage interface {
	// if not found will return empty string but no error (no special error
	// for NOT FOUND to be checked).
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, val []byte) error
}

type RedisKeyValStorage struct {
}

func NewRedisKeyValStorage() *RedisKeyValStorage {
	return &RedisKeyValStorage{}
}

func (s *RedisKeyValStorage) Get(ctx context.Context, key string) ([]byte, error) {
	// TODO
	return nil, fmt.Errorf("not implemented")
}

func (s *RedisKeyValStorage) Set(ctx context.Context, key string, val []byte) error {
	// TODO
	return fmt.Errorf("not implemented")
}

type NopKeyValStorage struct {
}

func NewNopKeyValStorage() *NopKeyValStorage {
	return &NopKeyValStorage{}
}

func (s *NopKeyValStorage) Get(ctx context.Context, key string) ([]byte, error) {
	// NOP :)
	return []byte{}, nil
}

func (s *NopKeyValStorage) Set(ctx context.Context, key string, val []byte) error {
	// NOP
	return nil
}

type InMemStorage struct {
	storage sync.Map
}

func NewInMemKeyValStorage() *InMemStorage {
	return &InMemStorage{}
}

func (s *InMemStorage) Get(ctx context.Context, key string) ([]byte, error) {
	val, ok := s.storage.Load(key)
	if ok {
		bval, bok := val.([]byte)
		if bok {
			return bval, nil
		}
	}
	return []byte{}, nil
}

func (s *InMemStorage) Set(ctx context.Context, key string, val []byte) error {
	s.storage.Store(key, val)
	return nil
}
