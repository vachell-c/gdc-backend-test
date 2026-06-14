package unit_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/vasti/gdc-backend-test/internal/handler"
	"github.com/vasti/gdc-backend-test/internal/middleware"
	"github.com/vasti/gdc-backend-test/internal/model"
	"github.com/vasti/gdc-backend-test/internal/service"
)

// Helper to create a TaskHandler with nil services.
// Only paths that don't touch the services (early returns) can be tested.
func newNilTaskHandler() *handler.TaskHandler {
	return handler.NewTaskHandler(nil, nil)
}

// Helper to create an Echo test context.
func newEchoContext(method, path, body string) (*echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	return c, rec
}

// setPathParam sets a named path parameter on an Echo v5 context.
func setPathParam(c *echo.Context, name, value string) {
	c.SetPathValues(echo.PathValues{{Name: name, Value: value}})
}

// assertAppError checks that an error is an *model.AppError with the given code.
func assertAppError(t *testing.T, err error, expectedCode string) {
	t.Helper()
	assert.Error(t, err)
	var appErr *model.AppError
	assert.ErrorAs(t, err, &appErr)
	if appErr != nil {
		assert.Equal(t, expectedCode, appErr.Code)
	}
}

// --- Create handler tests ---

func TestTaskHandler_Create_Unauthorized(t *testing.T) {
	h := newNilTaskHandler()
	c, _ := newEchoContext(http.MethodPost, "/tasks", `{"title":"Test Task"}`)

	err := h.Create(c)
	assertAppError(t, err, "ERR_UNAUTHORIZED")
}

func TestTaskHandler_Create_InvalidBody(t *testing.T) {
	h := newNilTaskHandler()
	c, _ := newEchoContext(http.MethodPost, "/tasks", `{invalid json}`)

	// Set user_id to pass the auth check
	c.Set("user_id", uuid.New())

	err := h.Create(c)
	assertAppError(t, err, "ERR_VALIDATION")
}

// --- List handler tests ---

func TestTaskHandler_List_Unauthorized(t *testing.T) {
	h := newNilTaskHandler()
	c, _ := newEchoContext(http.MethodGet, "/tasks", "")

	err := h.List(c)
	assertAppError(t, err, "ERR_UNAUTHORIZED")
}

// --- GetByID handler tests ---

func TestTaskHandler_GetByID_Unauthorized(t *testing.T) {
	h := newNilTaskHandler()
	c, _ := newEchoContext(http.MethodGet, "/tasks/123", "")

	err := h.GetByID(c)
	assertAppError(t, err, "ERR_UNAUTHORIZED")
}

func TestTaskHandler_GetByID_InvalidID(t *testing.T) {
	h := newNilTaskHandler()
	c, _ := newEchoContext(http.MethodGet, "/tasks/invalid-uuid", "")
	c.Set("user_id", uuid.New())
	setPathParam(c, "id", "not-a-uuid")

	err := h.GetByID(c)
	assertAppError(t, err, "ERR_VALIDATION")
}

// --- Update handler tests ---

func TestTaskHandler_Update_Unauthorized(t *testing.T) {
	h := newNilTaskHandler()
	c, _ := newEchoContext(http.MethodPut, "/tasks/123", `{"title":"Updated"}`)

	err := h.Update(c)
	assertAppError(t, err, "ERR_UNAUTHORIZED")
}

func TestTaskHandler_Update_InvalidID(t *testing.T) {
	h := newNilTaskHandler()
	c, _ := newEchoContext(http.MethodPut, "/tasks/invalid", `{"title":"Updated"}`)
	c.Set("user_id", uuid.New())
	setPathParam(c, "id", "not-a-uuid")

	err := h.Update(c)
	assertAppError(t, err, "ERR_VALIDATION")
}

func TestTaskHandler_Update_InvalidBody(t *testing.T) {
	h := newNilTaskHandler()
	c, _ := newEchoContext(http.MethodPut, "/tasks/valid-uuid", `{bad json}`)
	c.Set("user_id", uuid.New())
	setPathParam(c, "id", uuid.New().String())

	err := h.Update(c)
	assertAppError(t, err, "ERR_VALIDATION")
}

// --- Delete handler tests ---

func TestTaskHandler_Delete_Unauthorized(t *testing.T) {
	h := newNilTaskHandler()
	c, _ := newEchoContext(http.MethodDelete, "/tasks/123", "")

	err := h.Delete(c)
	assertAppError(t, err, "ERR_UNAUTHORIZED")
}

func TestTaskHandler_Delete_InvalidID(t *testing.T) {
	h := newNilTaskHandler()
	c, _ := newEchoContext(http.MethodDelete, "/tasks/invalid", "")
	c.Set("user_id", uuid.New())
	setPathParam(c, "id", "not-a-uuid")

	err := h.Delete(c)
	assertAppError(t, err, "ERR_VALIDATION")
}

// --- Assign handler tests ---

func TestTaskHandler_Assign_Unauthorized(t *testing.T) {
	h := newNilTaskHandler()
	c, _ := newEchoContext(http.MethodPost, "/tasks/123/assign", `{"user_id":"`+uuid.New().String()+`"}`)

	err := h.Assign(c)
	assertAppError(t, err, "ERR_UNAUTHORIZED")
}

func TestTaskHandler_Assign_InvalidID(t *testing.T) {
	h := newNilTaskHandler()
	c, _ := newEchoContext(http.MethodPost, "/tasks/invalid/assign", `{"user_id":"`+uuid.New().String()+`"}`)
	c.Set("user_id", uuid.New())
	setPathParam(c, "id", "not-a-uuid")

	err := h.Assign(c)
	assertAppError(t, err, "ERR_VALIDATION")
}

func TestTaskHandler_Assign_InvalidBody(t *testing.T) {
	h := newNilTaskHandler()
	c, _ := newEchoContext(http.MethodPost, "/tasks/valid-uuid/assign", `{bad}`)
	c.Set("user_id", uuid.New())
	setPathParam(c, "id", uuid.New().String())

	err := h.Assign(c)
	assertAppError(t, err, "ERR_VALIDATION")
}

// --- Middleware integration: GetUserID ---

func TestGetUserID_NotSet(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	uid := middleware.GetUserID(c)
	assert.Equal(t, uuid.Nil, uid)
	_ = rec
}

func TestGetUserID_Set(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	expected := uuid.New()
	c.Set("user_id", expected)

	uid := middleware.GetUserID(c)
	assert.Equal(t, expected, uid)
	_ = rec
}

func TestGetUserID_WrongType(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	c.Set("user_id", "not-a-uuid")

	uid := middleware.GetUserID(c)
	assert.Equal(t, uuid.Nil, uid)
	_ = rec
}

// --- Handler creation tests ---

func TestNewTaskHandler(t *testing.T) {
	h := handler.NewTaskHandler(nil, nil)
	assert.NotNil(t, h, "handler should be created even with nil services")
}

func TestNewTaskHandler_NilServices(t *testing.T) {
	h := handler.NewTaskHandler((*service.TaskService)(nil), (*service.TaskAssignService)(nil))
	assert.NotNil(t, h)
}
