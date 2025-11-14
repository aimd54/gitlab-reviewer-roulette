package mocks

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// MockCache is an in-memory mock implementation of the Cache interface
// Used for testing without requiring a real Redis instance
type MockCache struct {
	data map[string]interface{}
	sets map[string]map[string]bool
	mu   sync.RWMutex
}

// NewMockCache creates a new mock cache instance
func NewMockCache() *MockCache {
	return &MockCache{
		data: make(map[string]interface{}),
		sets: make(map[string]map[string]bool),
	}
}

// Get retrieves a value from the mock cache
func (m *MockCache) Get(ctx context.Context, key string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	val, exists := m.data[key]
	if !exists {
		return "", nil // Return empty string for non-existent keys (like Redis)
	}

	// Convert to string
	if strVal, ok := val.(string); ok {
		return strVal, nil
	}
	return "", nil
}

// Set stores a value in the mock cache
func (m *MockCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.data[key] = value
	// Note: expiration is ignored in mock (no TTL implementation)
	return nil
}

// Del deletes keys from the mock cache
func (m *MockCache) Del(ctx context.Context, keys ...string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, key := range keys {
		delete(m.data, key)
	}
	return nil
}

// Exists checks if keys exist in the mock cache
func (m *MockCache) Exists(ctx context.Context, keys ...string) (int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var count int64
	for _, key := range keys {
		if _, exists := m.data[key]; exists {
			count++
		}
	}
	return count, nil
}

// Incr increments a key's value
func (m *MockCache) Incr(ctx context.Context, key string) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	val, exists := m.data[key]
	if !exists {
		m.data[key] = "1"
		return 1, nil
	}

	// Try to parse as int
	var intVal int64
	if strVal, ok := val.(string); ok {
		_, err := fmt.Sscanf(strVal, "%d", &intVal)
		if err != nil {
			intVal = 0
		}
	}

	intVal++
	m.data[key] = fmt.Sprintf("%d", intVal)
	return intVal, nil
}

// Decr decrements a key's value
func (m *MockCache) Decr(ctx context.Context, key string) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	val, exists := m.data[key]
	if !exists {
		m.data[key] = "-1"
		return -1, nil
	}

	// Try to parse as int
	var intVal int64
	if strVal, ok := val.(string); ok {
		_, err := fmt.Sscanf(strVal, "%d", &intVal)
		if err != nil {
			intVal = 0
		}
	}

	intVal--
	m.data[key] = fmt.Sprintf("%d", intVal)
	return intVal, nil
}

// SAdd adds members to a set
func (m *MockCache) SAdd(ctx context.Context, key string, members ...interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.sets[key] == nil {
		m.sets[key] = make(map[string]bool)
	}

	for _, member := range members {
		memberStr := fmt.Sprintf("%v", member)
		m.sets[key][memberStr] = true
	}
	return nil
}

// SRem removes members from a set
func (m *MockCache) SRem(ctx context.Context, key string, members ...interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.sets[key] == nil {
		return nil
	}

	for _, member := range members {
		memberStr := fmt.Sprintf("%v", member)
		delete(m.sets[key], memberStr)
	}
	return nil
}

// SMembers returns all members of a set
func (m *MockCache) SMembers(ctx context.Context, key string) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	members := make([]string, 0)
	if m.sets[key] != nil {
		for member := range m.sets[key] {
			members = append(members, member)
		}
	}
	return members, nil
}

// SIsMember checks if a member exists in a set
func (m *MockCache) SIsMember(ctx context.Context, key string, member interface{}) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.sets[key] == nil {
		return false, nil
	}

	memberStr := fmt.Sprintf("%v", member)
	exists := m.sets[key][memberStr]
	return exists, nil
}

// SetNX sets a key only if it doesn't exist (for distributed locking)
func (m *MockCache) SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.data[key]; exists {
		return false, nil
	}

	m.data[key] = value
	return true, nil
}

// Expire sets an expiration on a key (no-op in mock)
func (m *MockCache) Expire(ctx context.Context, key string, expiration time.Duration) error {
	// No-op in mock (no TTL implementation)
	return nil
}

// Health always returns nil for mock
func (m *MockCache) Health(ctx context.Context) error {
	return nil
}

// Close is a no-op for mock
func (m *MockCache) Close() error {
	return nil
}

// Clear resets the mock cache (useful for tests)
func (m *MockCache) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.data = make(map[string]interface{})
	m.sets = make(map[string]map[string]bool)
}
