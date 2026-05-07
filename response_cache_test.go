package openrouter

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCreateChatCompletionResponseCacheHeadersAndMetadata(t *testing.T) {
	t.Parallel()

	enabled := true
	ttl := 3600
	httpClient := &fakeHTTPClient{
		response: responseCacheJSONResponse(`{
			"id":"chatcmpl_1",
			"object":"chat.completion",
			"model":"openai/gpt-4o-mini",
			"choices":[{"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}]
		}`),
	}
	client := newResponseCacheTestClient(httpClient)

	resp, err := client.CreateChatCompletion(context.Background(), ChatCompletionRequest{
		Model: "openai/gpt-4o-mini",
		Messages: []ChatCompletionMessage{
			UserMessage("hello"),
		},
		ResponseCache: &ResponseCacheConfig{
			Enabled:    &enabled,
			TTLSeconds: &ttl,
			Clear:      true,
		},
	})

	require.NoError(t, err)
	requireResponseCacheRequestHeaders(t, httpClient.lastRequest, "true", "3600", "true")
	requireRequestBodyOmitsResponseCache(t, httpClient.lastRequest)
	require.NotNil(t, resp.ResponseCache)
	require.Equal(t, ResponseCacheStatusHit, resp.ResponseCache.Status)
	require.NotNil(t, resp.ResponseCache.AgeSeconds)
	require.Equal(t, 42, *resp.ResponseCache.AgeSeconds)
	require.NotNil(t, resp.ResponseCache.TTLSeconds)
	require.Equal(t, 3600, *resp.ResponseCache.TTLSeconds)
}

func TestCreateCompletionResponseCacheHeadersAndMetadata(t *testing.T) {
	t.Parallel()

	enabled := false
	httpClient := &fakeHTTPClient{
		response: responseCacheJSONResponse(`{
			"id":"cmpl_1",
			"object":"text_completion",
			"model":"openai/gpt-4o-mini",
			"choices":[{"text":"ok","finish_reason":"stop"}]
		}`),
	}
	client := newResponseCacheTestClient(httpClient)

	resp, err := client.CreateCompletion(context.Background(), CompletionRequest{
		Model:  "openai/gpt-4o-mini",
		Prompt: "hello",
		ResponseCache: &ResponseCacheConfig{
			Enabled: &enabled,
		},
	})

	require.NoError(t, err)
	requireResponseCacheRequestHeaders(t, httpClient.lastRequest, "false", "", "")
	requireRequestBodyOmitsResponseCache(t, httpClient.lastRequest)
	require.NotNil(t, resp.ResponseCache)
	require.Equal(t, ResponseCacheStatusHit, resp.ResponseCache.Status)
}

func TestCreateEmbeddingsResponseCacheHeadersAndMetadata(t *testing.T) {
	t.Parallel()

	ttl := 120
	httpClient := &fakeHTTPClient{
		response: responseCacheJSONResponse(`{
			"id":"embd_1",
			"object":"list",
			"data":[{"object":"embedding","embedding":[0.1,0.2],"index":0}],
			"model":"test-embeddings-model"
		}`),
	}
	client := newResponseCacheTestClient(httpClient)

	resp, err := client.CreateEmbeddings(context.Background(), EmbeddingsRequest{
		Model: "test-embeddings-model",
		Input: "hello",
		ResponseCache: &ResponseCacheConfig{
			TTLSeconds: &ttl,
		},
	})

	require.NoError(t, err)
	requireResponseCacheRequestHeaders(t, httpClient.lastRequest, "", "120", "")
	requireRequestBodyOmitsResponseCache(t, httpClient.lastRequest)
	require.NotNil(t, resp.ResponseCache)
	require.Equal(t, ResponseCacheStatusHit, resp.ResponseCache.Status)
}

func TestResponseCacheClearEnablesCacheWhenUnset(t *testing.T) {
	t.Parallel()

	httpClient := &fakeHTTPClient{
		response: responseCacheJSONResponse(`{
			"id":"embd_1",
			"object":"list",
			"data":[{"object":"embedding","embedding":[0.1,0.2],"index":0}],
			"model":"test-embeddings-model"
		}`),
	}
	client := newResponseCacheTestClient(httpClient)

	_, err := client.CreateEmbeddings(context.Background(), EmbeddingsRequest{
		Model: "test-embeddings-model",
		Input: "hello",
		ResponseCache: &ResponseCacheConfig{
			Clear: true,
		},
	})

	require.NoError(t, err)
	requireResponseCacheRequestHeaders(t, httpClient.lastRequest, "true", "", "true")
}

func TestCreateChatCompletionStreamResponseCacheHeadersAndMetadata(t *testing.T) {
	t.Parallel()

	enabled := true
	httpClient := &fakeHTTPClient{
		response: responseCacheStreamResponse(strings.Join([]string{
			`data: {"id":"chatcmpl_1","model":"openai/gpt-4o-mini","choices":[{"delta":{"content":"ok"}}]}`,
			`data: [DONE]`,
			``,
		}, "\n")),
	}
	client := newResponseCacheTestClient(httpClient)

	stream, err := client.CreateChatCompletionStream(context.Background(), ChatCompletionRequest{
		Model:         "openai/gpt-4o-mini",
		Messages:      []ChatCompletionMessage{UserMessage("hello")},
		ResponseCache: &ResponseCacheConfig{Enabled: &enabled},
	})
	require.NoError(t, err)
	defer stream.Close()

	requireResponseCacheRequestHeaders(t, httpClient.lastRequest, "true", "", "")
	metadata := stream.ResponseCacheMetadata()
	require.NotNil(t, metadata)
	require.Equal(t, ResponseCacheStatusHit, metadata.Status)
}

func TestCreateCompletionStreamResponseCacheHeadersAndMetadata(t *testing.T) {
	t.Parallel()

	ttl := 60
	httpClient := &fakeHTTPClient{
		response: responseCacheStreamResponse(strings.Join([]string{
			`data: {"id":"cmpl_1","model":"openai/gpt-4o-mini","choices":[{"text":"ok"}]}`,
			`data: [DONE]`,
			``,
		}, "\n")),
	}
	client := newResponseCacheTestClient(httpClient)

	stream, err := client.CreateCompletionStream(context.Background(), CompletionRequest{
		Model:         "openai/gpt-4o-mini",
		Prompt:        "hello",
		ResponseCache: &ResponseCacheConfig{TTLSeconds: &ttl},
	})
	require.NoError(t, err)
	defer stream.Close()

	requireResponseCacheRequestHeaders(t, httpClient.lastRequest, "", "60", "")
	metadata := stream.ResponseCacheMetadata()
	require.NotNil(t, metadata)
	require.Equal(t, ResponseCacheStatusHit, metadata.Status)
}

func newResponseCacheTestClient(httpClient HTTPDoer) *Client {
	cfg := DefaultConfig("test-token")
	cfg.BaseURL = "https://example.com/api/v1"
	cfg.HTTPClient = httpClient
	return NewClientWithConfig(*cfg)
}

func requireResponseCacheRequestHeaders(
	t *testing.T,
	req *http.Request,
	enabled string,
	ttl string,
	clear string,
) {
	t.Helper()

	require.NotNil(t, req)
	require.Equal(t, enabled, req.Header.Get(headerResponseCache))
	require.Equal(t, ttl, req.Header.Get(headerResponseCacheTTL))
	require.Equal(t, clear, req.Header.Get(headerResponseCacheClear))
}

func requireRequestBodyOmitsResponseCache(t *testing.T, req *http.Request) {
	t.Helper()

	body, err := io.ReadAll(req.Body)
	require.NoError(t, err)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(body, &payload))
	require.NotContains(t, payload, "ResponseCache")
	require.NotContains(t, payload, "response_cache")
}

func responseCacheJSONResponse(body string) *http.Response {
	resp := jsonResponse(http.StatusOK, body)
	addResponseCacheHeaders(resp.Header)
	return resp
}

func responseCacheStreamResponse(body string) *http.Response {
	resp := jsonResponse(http.StatusOK, body)
	addResponseCacheHeaders(resp.Header)
	return resp
}

func addResponseCacheHeaders(header http.Header) {
	header.Set(headerResponseCacheStatus, string(ResponseCacheStatusHit))
	header.Set(headerResponseCacheAge, "42")
	header.Set(headerResponseCacheTTL, "3600")
}
