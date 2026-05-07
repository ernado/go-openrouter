package openrouter

import (
	"net/http"
	"strconv"
)

const (
	headerResponseCache       = "X-OpenRouter-Cache"
	headerResponseCacheTTL    = "X-OpenRouter-Cache-TTL"
	headerResponseCacheClear  = "X-OpenRouter-Cache-Clear"
	headerResponseCacheStatus = "X-OpenRouter-Cache-Status"
	headerResponseCacheAge    = "X-OpenRouter-Cache-Age"
)

const (
	ResponseCacheStatusHit  ResponseCacheStatus = "HIT"
	ResponseCacheStatusMiss ResponseCacheStatus = "MISS"
)

// ResponseCacheStatus is the cache result returned by OpenRouter.
type ResponseCacheStatus string

// ResponseCacheConfig controls OpenRouter response caching headers for a request.
type ResponseCacheConfig struct {
	// Enabled sets X-OpenRouter-Cache to true or false when provided.
	Enabled *bool
	// TTLSeconds sets X-OpenRouter-Cache-TTL when provided.
	// OpenRouter accepts values from 1 to 86400 seconds.
	TTLSeconds *int
	// Clear sets X-OpenRouter-Cache-Clear to true.
	Clear bool
}

// ResponseCacheMetadata contains OpenRouter response cache headers returned by the API.
type ResponseCacheMetadata struct {
	Status     ResponseCacheStatus
	AgeSeconds *int
	TTLSeconds *int
}

func setResponseCacheHeaders(header http.Header, cache *ResponseCacheConfig) {
	if cache == nil {
		return
	}
	if cache.Enabled != nil {
		header.Set(headerResponseCache, strconv.FormatBool(*cache.Enabled))
	} else if cache.Clear {
		header.Set(headerResponseCache, "true")
	}
	if cache.TTLSeconds != nil {
		header.Set(headerResponseCacheTTL, strconv.Itoa(*cache.TTLSeconds))
	}
	if cache.Clear {
		header.Set(headerResponseCacheClear, "true")
	}
}

func setResponseCacheHeadersFromBody(header http.Header, body any) {
	switch request := body.(type) {
	case ChatCompletionRequest:
		setResponseCacheHeaders(header, request.ResponseCache)
	case *ChatCompletionRequest:
		if request != nil {
			setResponseCacheHeaders(header, request.ResponseCache)
		}
	case CompletionRequest:
		setResponseCacheHeaders(header, request.ResponseCache)
	case *CompletionRequest:
		if request != nil {
			setResponseCacheHeaders(header, request.ResponseCache)
		}
	case EmbeddingsRequest:
		setResponseCacheHeaders(header, request.ResponseCache)
	case *EmbeddingsRequest:
		if request != nil {
			setResponseCacheHeaders(header, request.ResponseCache)
		}
	}
}

func parseResponseCacheMetadata(header http.Header) *ResponseCacheMetadata {
	status := header.Get(headerResponseCacheStatus)
	age := parseHeaderInt(header, headerResponseCacheAge)
	ttl := parseHeaderInt(header, headerResponseCacheTTL)
	if status == "" && age == nil && ttl == nil {
		return nil
	}
	return &ResponseCacheMetadata{
		Status:     ResponseCacheStatus(status),
		AgeSeconds: age,
		TTLSeconds: ttl,
	}
}

func parseHeaderInt(header http.Header, key string) *int {
	value := header.Get(key)
	if value == "" {
		return nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return nil
	}
	return &parsed
}

func setResponseCacheMetadata(v any, header http.Header) {
	metadata := parseResponseCacheMetadata(header)
	if metadata == nil {
		return
	}
	switch response := v.(type) {
	case *ChatCompletionResponse:
		response.ResponseCache = metadata
	case *CompletionResponse:
		response.ResponseCache = metadata
	case *EmbeddingsResponse:
		response.ResponseCache = metadata
	}
}
