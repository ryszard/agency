package client

import (
	"context"
	"testing"

	"github.com/ryszard/agency/util/cache"
)

type mockClient struct {
	called bool
}

type mockCache struct {
	hit   bool
	cache Cache
}

func (m *mockCache) Get(key []byte) ([]byte, bool, error) {
	v, ok, err := m.cache.Get(key)
	if err != nil {
		return nil, false, err
	}
	if !ok {
		return nil, false, nil
	}
	m.hit = true
	return v, ok, nil
}

// Set implements Cache
func (c *mockCache) Set(key []byte, value []byte) error {
	return c.cache.Set(key, value)
}

var _ Cache = &mockCache{}

func (m *mockClient) CreateChatCompletion(ctx context.Context, req ChatCompletionRequest) (ChatCompletionResponse, error) {
	m.called = true
	return ChatCompletionResponse{}, nil
}

func TestCachedClient(t *testing.T) {
	mockC := &mockClient{}
	mockCache := &mockCache{cache: cache.Memory()}

	cachedC := &CachedClient{
		client: mockC,
		cache:  mockCache,
	}

	ctx := context.Background()
	req := ChatCompletionRequest{} // Add properties as needed

	// First call, the underlying client should be called and the result cached
	cachedC.CreateChatCompletion(ctx, req)
	if !mockC.called {
		t.Fatalf("Expected underlying client to be called on first request")
	}
	if mockCache.hit {
		t.Fatalf("Expected cache not to be hit on first request")
	}

	// Reset state
	mockC.called = false
	mockCache.hit = false

	// Second call with identical request, the cache should be hit and the underlying client should not be called
	cachedC.CreateChatCompletion(ctx, req)
	if mockC.called {
		t.Fatalf("Expected underlying client to not be called on second request")
	}
	if !mockCache.hit {
		t.Fatalf("Expected cache to be hit on second request")
	}
}
