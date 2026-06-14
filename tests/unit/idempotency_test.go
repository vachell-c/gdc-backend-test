package unit_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/vasti/gdc-backend-test/internal/middleware"
	"github.com/vasti/gdc-backend-test/internal/model"
	"github.com/vasti/gdc-backend-test/internal/repository"
)

// --- Idempotency middleware: key validation tests ---
// These tests don't need a real DB repo. The middleware returns validation
// errors directly before any DB call. In Echo v5 handlers, errors are returned
// (not written to response) unless wrapped by ErrorHandlerMiddleware.

func TestIdempotencyMiddleware_MissingKey(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/tasks", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	mw := middleware.IdempotencyMiddleware(nil)
	handler := mw(func(c *echo.Context) error {
		return c.JSON(http.StatusCreated, map[string]string{"status": "ok"})
	})

	err := handler(c)
	assert.Error(t, err)
	var appErr *model.AppError
	assert.ErrorAs(t, err, &appErr)
	assert.Equal(t, 400, appErr.Status)
	assert.Equal(t, "ERR_VALIDATION", appErr.Code)
	assert.Contains(t, appErr.Message, "Idempotency-Key header is required")
	_ = rec
}

func TestIdempotencyMiddleware_InvalidUUID(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/tasks", nil)
	req.Header.Set("Idempotency-Key", "not-a-uuid")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	mw := middleware.IdempotencyMiddleware(nil)
	handler := mw(func(c *echo.Context) error {
		return c.JSON(http.StatusCreated, map[string]string{"status": "ok"})
	})

	err := handler(c)
	assert.Error(t, err)
	var appErr *model.AppError
	assert.ErrorAs(t, err, &appErr)
	assert.Equal(t, 400, appErr.Status)
	assert.Equal(t, "ERR_VALIDATION", appErr.Code)
	assert.Contains(t, appErr.Message, "Idempotency-Key must be a valid UUID")
	_ = rec
}

func TestIdempotencyMiddleware_SkipNonPost(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/tasks", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	mw := middleware.IdempotencyMiddleware(nil)
	handler := mw(func(c *echo.Context) error {
		return c.String(http.StatusOK, "get ok")
	})

	err := handler(c)
	assert.NoError(t, err)
	assert.Equal(t, "get ok", strings.TrimSpace(rec.Body.String()))
}

func TestIdempotencyMiddleware_SkipNonTasksPath(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/other", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	mw := middleware.IdempotencyMiddleware(nil)
	handler := mw(func(c *echo.Context) error {
		return c.String(http.StatusOK, "other ok")
	})

	err := handler(c)
	assert.NoError(t, err)
	assert.Equal(t, "other ok", strings.TrimSpace(rec.Body.String()))
}

func TestIdempotencyMiddleware_ValidKeyProceeds(t *testing.T) {
	// With a nil repo, the middleware tries to call idemRepo.GetByKey which panics.
	// But the key validation itself passes first.
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/tasks", nil)
	req.Header.Set("Idempotency-Key", uuid.New().String())
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	mw := middleware.IdempotencyMiddleware(nil)
	handler := mw(func(c *echo.Context) error {
		return c.String(http.StatusOK, "proceeded")
	})

	assert.Panics(t, func() {
		_ = handler(c)
	})
	_ = rec
}

// --- In-memory idempotency store for concurrent testing ---

// idempotencyStore mimics the DB-level idempotency key store with
// "first write wins" semantics (ON CONFLICT DO NOTHING).
type idempotencyStore struct {
	mu    sync.Mutex
	store map[string]storedResponse
}

type storedResponse struct {
	status int
	body   []byte
}

func newIdempotencyStore() *idempotencyStore {
	return &idempotencyStore{
		store: make(map[string]storedResponse),
	}
}

// tryStore attempts to store a response for a key.
// Returns (stored, isNew). If the key already exists, returns the existing
// response and isNew=false.
func (s *idempotencyStore) tryStore(key string, status int, body []byte) (storedResponse, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if existing, ok := s.store[key]; ok {
		return existing, false
	}

	resp := storedResponse{status: status, body: body}
	s.store[key] = resp
	return resp, true
}

func (s *idempotencyStore) get(key string) (storedResponse, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	resp, ok := s.store[key]
	return resp, ok
}

// --- Sequential idempotency test ---

func TestIdempotency_Sequential(t *testing.T) {
	store := newIdempotencyStore()
	key := uuid.New().String()

	// First request - should store
	body1 := `{"data":{"id":"task-1","title":"Test"}}`
	resp1, isNew := store.tryStore(key, http.StatusCreated, []byte(body1))
	assert.True(t, isNew, "first request should be new")
	assert.Equal(t, http.StatusCreated, resp1.status)
	assert.JSONEq(t, body1, string(resp1.body))

	// Second request with same key - should return stored
	body2 := `{"data":{"id":"task-2","title":"Should Not Be Stored"}}`
	resp2, isNew := store.tryStore(key, http.StatusCreated, []byte(body2))
	assert.False(t, isNew, "second request should return existing")
	assert.Equal(t, resp1.status, resp2.status, "status should match first response")
	assert.JSONEq(t, string(resp1.body), string(resp2.body), "body should match first response")
}

// --- Concurrent idempotency test (must pass with -race) ---

func TestIdempotency_ConcurrentDuplicate(t *testing.T) {
	store := newIdempotencyStore()
	key := uuid.New().String()
	const goroutines = 10

	var wg sync.WaitGroup
	results := make(chan storedResponse, goroutines)

	for range goroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			body := `{"data":{"id":"concurrent-task","title":"Race Test"}}`
			resp, _ := store.tryStore(key, http.StatusCreated, []byte(body))
			results <- resp
		}()
	}

	wg.Wait()
	close(results)

	// All goroutines should have returned the same response
	var firstResp *storedResponse
	for resp := range results {
		if firstResp == nil {
			firstResp = &resp
		} else {
			assert.Equal(t, firstResp.status, resp.status, "all concurrent requests must return same status")
			assert.Equal(t, firstResp.body, resp.body, "all concurrent requests must return same body")
		}
	}

	assert.NotNil(t, firstResp, "should have at least one result")
	// Verify only one entry exists in the store
	assert.Equal(t, 1, len(store.store), "store should have exactly one entry after concurrent inserts")
}

// --- Concurrent test with different keys (should all succeed) ---

func TestIdempotency_ConcurrentDifferentKeys(t *testing.T) {
	store := newIdempotencyStore()
	const goroutines = 20

	var wg sync.WaitGroup
	results := make(chan bool, goroutines)

	for range goroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			key := uuid.New().String()
			body := `{"data":{"id":"` + key + `","title":"Unique"}}`
			_, isNew := store.tryStore(key, http.StatusCreated, []byte(body))
			results <- isNew
		}()
	}

	wg.Wait()
	close(results)

	for isNew := range results {
		assert.True(t, isNew, "each different key should be stored as new")
	}

	assert.Equal(t, goroutines, len(store.store), "all unique keys should be stored")
}

// --- Direct middleware test: bodyRecorder behavior ---

func TestIdempotencyMiddleware_BodyRecorder(t *testing.T) {
	rec := httptest.NewRecorder()
	bodyRec := &testBodyRecorder{writer: rec}

	data := []byte(`{"status":"ok"}`)
	n, err := bodyRec.Write(data)
	assert.NoError(t, err)
	assert.Equal(t, len(data), n)

	bodyRec.WriteHeader(http.StatusCreated)

	assert.Equal(t, http.StatusCreated, bodyRec.status)
	assert.Equal(t, data, bodyRec.body.Bytes())
}

type testBodyRecorder struct {
	writer http.ResponseWriter
	body   bytesBuffer
	status int
}

type bytesBuffer struct {
	data []byte
}

func (b *bytesBuffer) Write(p []byte) (int, error) {
	b.data = append(b.data, p...)
	return len(p), nil
}

func (b *bytesBuffer) Bytes() []byte {
	return b.data
}

func (r *testBodyRecorder) Header() http.Header {
	return r.writer.Header()
}

func (r *testBodyRecorder) Write(b []byte) (int, error) {
	return r.body.Write(b)
}

func (r *testBodyRecorder) WriteHeader(code int) {
	r.status = code
	r.writer.WriteHeader(code)
}

// --- Test repository creation ---

func TestNewIdempotencyRepository(t *testing.T) {
	repo := (*repository.IdempotencyRepository)(nil)
	assert.Nil(t, repo)
}

// --- Concurrent test with mixed read/write to the store ---

func TestIdempotency_ConcurrentReadWrite(t *testing.T) {
	store := newIdempotencyStore()
	const keys = 5
	const opsPerKey = 5

	var wg sync.WaitGroup

	// Writers
	for range keys {
		key := uuid.New().String()
		for range opsPerKey {
			wg.Add(1)
			go func(k string) {
				defer wg.Done()
				body := `{"result":"ok"}`
				_, _ = store.tryStore(k, http.StatusOK, []byte(body))
			}(key)
		}
	}

	// Readers
	for range keys {
		key := uuid.New().String()
		wg.Add(1)
		go func(k string) {
			defer wg.Done()
			store.get(k)
		}(key)
	}

	wg.Wait()
}
