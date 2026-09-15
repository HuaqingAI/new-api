package aionui

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/middleware"
	agentplatform "github.com/QuantumNous/new-api/service/agentplatform"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type clientLogArtifactStore struct {
	files map[string][]byte
	fail  bool
}

func (s *clientLogArtifactStore) PutFile(_ context.Context, input agentplatform.PutArtifactInput) (agentplatform.ArtifactRef, error) {
	if s.fail {
		return agentplatform.ArtifactRef{}, os.ErrPermission
	}
	data, err := os.ReadFile(input.LocalPath)
	if err != nil {
		return agentplatform.ArtifactRef{}, err
	}
	s.files[input.BucketKey] = data
	return agentplatform.ArtifactRef{Key: input.BucketKey, Sha256: input.Sha256, Size: input.SizeBytes}, nil
}

func (s *clientLogArtifactStore) PresignPut(context.Context, agentplatform.PutArtifactInput, time.Duration) (agentplatform.PresignedArtifact, agentplatform.ArtifactRef, error) {
	return agentplatform.PresignedArtifact{}, agentplatform.ArtifactRef{}, io.EOF
}

func (s *clientLogArtifactStore) PresignGet(context.Context, agentplatform.ArtifactRef, time.Duration) (agentplatform.PresignedArtifact, error) {
	return agentplatform.PresignedArtifact{}, io.EOF
}

func (s *clientLogArtifactStore) DownloadToFile(context.Context, agentplatform.ArtifactRef, string) error {
	return io.EOF
}

func (s *clientLogArtifactStore) Delete(context.Context, agentplatform.ArtifactRef) error {
	return nil
}

func TestUploadClientLogs(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &clientLogArtifactStore{files: make(map[string][]byte)}
	restoreStore := agentplatform.SetArtifactStoreForTest(store)
	defer restoreStore()

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("aionui_user_id", 7)
		c.Set("aionui_email", "Alice@example.com")
		c.Next()
	})
	router.POST("/client-logs/:date", UploadClientLogs)

	t.Run("uploads every safe filename to its deterministic object key", func(t *testing.T) {
		response := performClientLogUpload(t, router, "2026-09-10", "0.1.1", map[string]string{
			"2026-09-10.log":          "desktop log",
			"2026-09-10.aioncore.log": "backend log",
			"trace.json":              "{}",
		})
		require.Equal(t, http.StatusOK, response.Code)
		assert.Contains(t, store.files, "client-logs/alice@example.com/0.1.1/2026/09/10/2026-09-10.log")
		assert.Contains(t, store.files, "client-logs/alice@example.com/0.1.1/2026/09/10/2026-09-10.aioncore.log")
		assert.Contains(t, store.files, "client-logs/alice@example.com/0.1.1/2026/09/10/trace.json")
	})

	t.Run("overwrites an existing key on a later upload", func(t *testing.T) {
		response := performClientLogUpload(t, router, "2026-09-10", "0.1.1", map[string]string{"2026-09-10.log": "first"})
		require.Equal(t, http.StatusOK, response.Code)
		response = performClientLogUpload(t, router, "2026-09-10", "0.1.1", map[string]string{"2026-09-10.log": "second"})
		require.Equal(t, http.StatusOK, response.Code)
		assert.Equal(t, []byte("second"), store.files["client-logs/alice@example.com/0.1.1/2026/09/10/2026-09-10.log"])
	})

	for name, request := range map[string]func() *http.Request{
		"invalid date": func() *http.Request {
			return newClientLogUploadRequest(t, "2026-02-30", "0.1.1", map[string]string{"log.txt": "body"})
		},
		"path traversal filename": func() *http.Request {
			return newClientLogUploadRequest(t, "2026-09-10", "0.1.1", map[string]string{"../log.txt": "body"})
		},
		"invalid version": func() *http.Request {
			return newClientLogUploadRequest(t, "2026-09-10", "../0.1.1", map[string]string{"log.txt": "body"})
		},
		"unknown multipart field": func() *http.Request {
			var body bytes.Buffer
			writer := multipart.NewWriter(&body)
			require.NoError(t, writer.WriteField("unexpected", "body"))
			require.NoError(t, writer.Close())
			request := httptest.NewRequest(http.MethodPost, "/client-logs/2026-09-10", &body)
			request.Header.Set("Content-Type", writer.FormDataContentType())
			request.Header.Set("X-AionUI-Client-Version", "0.1.1")
			return request
		},
		"empty file": func() *http.Request {
			return newClientLogUploadRequest(t, "2026-09-10", "0.1.1", map[string]string{"empty.log": ""})
		},
		"control character filename": func() *http.Request {
			return newClientLogUploadRequest(t, "2026-09-10", "0.1.1", map[string]string{"invalid\x01.log": "body"})
		},
	} {
		t.Run(name, func(t *testing.T) {
			before := len(store.files)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request())
			assert.Equal(t, http.StatusBadRequest, response.Code)
			assert.Len(t, store.files, before)
		})
	}

	t.Run("rejects duplicate multipart filenames without writing an object", func(t *testing.T) {
		before := len(store.files)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, newClientLogUploadPartsRequest(t, "2026-09-10", "0.1.1", [][2]string{{"same.log", "first"}, {"same.log", "second"}}))
		assert.Equal(t, http.StatusBadRequest, response.Code)
		assert.Len(t, store.files, before)
	})

	t.Run("rejects non-multipart input without writing an object", func(t *testing.T) {
		before := len(store.files)
		request := httptest.NewRequest(http.MethodPost, "/client-logs/2026-09-10", strings.NewReader("{}"))
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("X-AionUI-Client-Version", "0.1.1")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		assert.Equal(t, http.StatusUnsupportedMediaType, response.Code)
		assert.Len(t, store.files, before)
	})

	t.Run("does not expose store errors", func(t *testing.T) {
		store.fail = true
		defer func() { store.fail = false }()
		response := performClientLogUpload(t, router, "2026-09-10", "0.1.1", map[string]string{"log.txt": "body"})
		assert.Equal(t, http.StatusBadGateway, response.Code)
		assert.NotContains(t, response.Body.String(), "permission")
	})
}

func TestAionUiClientLogUploadRateLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	redisEnabled := common.RedisEnabled
	common.RedisEnabled = false
	defer func() { common.RedisEnabled = redisEnabled }()

	router := gin.New()
	userID := int(time.Now().UnixNano()) + 1
	router.Use(func(c *gin.Context) {
		c.Set("aionui_user_id", userID)
		c.Set("aionui_device_id", "test-device")
		c.Next()
	})
	router.Use(middleware.AionUiClientLogUploadRateLimit())
	router.POST("/client-logs", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	for range 3 {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/client-logs", nil))
		require.Equal(t, http.StatusNoContent, response.Code)
	}
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/client-logs", nil))
	require.Equal(t, http.StatusTooManyRequests, response.Code)
	assert.Equal(t, "900", response.Header().Get("Retry-After"))
}

func TestAionUiClientLogUploadRateLimitFallsBackWhenRedisUnavailable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	redisEnabled := common.RedisEnabled
	rdb := common.RDB
	common.RedisEnabled = true
	common.RDB = nil
	defer func() {
		common.RedisEnabled = redisEnabled
		common.RDB = rdb
	}()

	router := gin.New()
	userID := int(time.Now().UnixNano()) + 2
	router.Use(func(c *gin.Context) {
		c.Set("aionui_user_id", userID)
		c.Set("aionui_device_id", "redis-fallback-device")
		c.Next()
	})
	router.Use(middleware.AionUiClientLogUploadRateLimit())
	router.POST("/client-logs", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	for range 3 {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/client-logs", nil))
		require.Equal(t, http.StatusNoContent, response.Code)
	}
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/client-logs", nil))
	require.Equal(t, http.StatusTooManyRequests, response.Code)
}

func performClientLogUpload(t *testing.T, router http.Handler, date string, version string, files map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	response := httptest.NewRecorder()
	router.ServeHTTP(response, newClientLogUploadRequest(t, date, version, files))
	return response
}

func newClientLogUploadRequest(t *testing.T, date string, version string, files map[string]string) *http.Request {
	pairs := make([][2]string, 0, len(files))
	for name, content := range files {
		pairs = append(pairs, [2]string{name, content})
	}
	return newClientLogUploadPartsRequest(t, date, version, pairs)
}

func newClientLogUploadPartsRequest(t *testing.T, date string, version string, files [][2]string) *http.Request {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for _, file := range files {
		part, err := writer.CreateFormFile("files", file[0])
		require.NoError(t, err)
		_, err = part.Write([]byte(file[1]))
		require.NoError(t, err)
	}
	require.NoError(t, writer.Close())
	request := httptest.NewRequest(http.MethodPost, "/client-logs/"+date, &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	request.Header.Set("X-AionUI-Client-Version", version)
	return request
}

var _ agentplatform.ArtifactStore = (*clientLogArtifactStore)(nil)
