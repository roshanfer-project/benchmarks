package utils

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
)

var (
	// store replaces Redis for in-memory storage
	store sync.Map
)

type SGVertex struct {
	UserId    string   `json:"userId"`
	Followers []string `json:"followers"`
	Followees []string `json:"followees"` // Fixed to match the JSON key
}

var logg = GetLogger("state")

func GetState[T interface{}](ctx context.Context, key string) (T, error) {
	var value T

	// Retrieve state directly from Memory
	result, ok := store.Load(key)
	if !ok {
		// Key does not exist
		logg.Debug("[GetState] Key not found", "key", key)
		return value, errors.New("key not found")
	}

	resultBytes := result.([]byte)
	logg.Debug("[GetState] Retrieved raw value from Memory", "key", key, "value", string(resultBytes))

	// Unmarshal the JSON-encoded value into the value of type T
	err := json.Unmarshal(resultBytes, &value)
	if err != nil {
		logg.Error("[GetState] Error unmarshalling value", "key", key, "error", err)
		return value, err
	}
	logg.Debug("[GetState] Retrieved unmarshaled value for key", "key", key, "value", value)
	return value, nil
}

func GetBulkState[T interface{}](ctx context.Context, keys []string) ([]T, error) {
	var returnValues []T
	for _, key := range keys {
		val, err := GetState[T](ctx, key)
		if err != nil {
			return nil, fmt.Errorf("key %s not found", key)
		}
		returnValues = append(returnValues, val)
	}
	return returnValues, nil
}

func GetBulkStateDefault[T interface{}](ctx context.Context, keys []string, defVal T) ([]T, error) {
	var returnValues []T
	for _, key := range keys {
		val, err := GetState[T](ctx, key)
		if err != nil {
			val = defVal
		}
		returnValues = append(returnValues, val)
	}
	return returnValues, nil
}

func SetState(ctx context.Context, key string, value interface{}) error {
	valueBytes, err := json.Marshal(value)
	if err != nil {
		logg.Error("[SetState] Error marshalling value", "key", key, "error", err)
		return err
	}

	// Save state directly to Memory
	store.Store(key, valueBytes)

	logg.Debug("[SetState] Set value for key", "key", key, "value", value)
	return nil
}

func SetBulkState(ctx context.Context, kvs map[string]interface{}) error {
	for k, v := range kvs {
		err := SetState(ctx, k, v)
		if err != nil {
			return err
		}
	}
	return nil
}
