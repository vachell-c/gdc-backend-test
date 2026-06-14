package handler

import (
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
	"github.com/vasti/gdc-backend-test/internal/middleware"
	"github.com/vasti/gdc-backend-test/internal/model"
	"github.com/vasti/gdc-backend-test/internal/service"
)

// TaskHandler handles task CRUD and assignment HTTP requests.
type TaskHandler struct {
	taskSvc       *service.TaskService
	taskAssignSvc *service.TaskAssignService
}

// NewTaskHandler creates a new TaskHandler.
func NewTaskHandler(taskSvc *service.TaskService, taskAssignSvc *service.TaskAssignService) *TaskHandler {
	return &TaskHandler{
		taskSvc:       taskSvc,
		taskAssignSvc: taskAssignSvc,
	}
}

// Create creates a new task for the authenticated user.
func (h *TaskHandler) Create(c *echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		return model.ErrUnauthorized("user not authenticated", nil)
	}

	var req model.CreateTaskRequest
	if err := c.Bind(&req); err != nil {
		return model.ErrValidation("invalid request body", err)
	}

	task, err := h.taskSvc.Create(c.Request().Context(), userID, req)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusCreated, model.TaskResponse{Data: *task})
}

// List retrieves paginated tasks for the authenticated user.
func (h *TaskHandler) List(c *echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		return model.ErrUnauthorized("user not authenticated", nil)
	}

	status := c.QueryParam("status")
	search := c.QueryParam("search")

	pageStr := c.QueryParam("page")
	page := 1
	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	limitStr := c.QueryParam("limit")
	limit := 10
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	result, err := h.taskSvc.List(c.Request().Context(), userID, status, search, page, limit)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, result)
}

// GetByID retrieves a single task by ID.
func (h *TaskHandler) GetByID(c *echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		return model.ErrUnauthorized("user not authenticated", nil)
	}

	taskID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return model.ErrValidation("invalid task id", err)
	}

	task, err := h.taskSvc.GetByID(c.Request().Context(), userID, taskID)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, model.TaskResponse{Data: *task})
}

// Update updates an existing task.
func (h *TaskHandler) Update(c *echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		return model.ErrUnauthorized("user not authenticated", nil)
	}

	taskID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return model.ErrValidation("invalid task id", err)
	}

	var req model.UpdateTaskRequest
	if err := c.Bind(&req); err != nil {
		return model.ErrValidation("invalid request body", err)
	}

	task, err := h.taskSvc.Update(c.Request().Context(), userID, taskID, req)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, model.TaskResponse{Data: *task})
}

// Delete removes a task.
func (h *TaskHandler) Delete(c *echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		return model.ErrUnauthorized("user not authenticated", nil)
	}

	taskID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return model.ErrValidation("invalid task id", err)
	}

	if err := h.taskSvc.Delete(c.Request().Context(), userID, taskID); err != nil {
		return err
	}

	return c.NoContent(http.StatusNoContent)
}

// Assign assigns a task to another user in the same team.
func (h *TaskHandler) Assign(c *echo.Context) error {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		return model.ErrUnauthorized("user not authenticated", nil)
	}

	taskID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return model.ErrValidation("invalid task id", err)
	}

	var req model.AssignRequest
	if err := c.Bind(&req); err != nil {
		return model.ErrValidation("invalid request body", err)
	}

	task, err := h.taskAssignSvc.Assign(c.Request().Context(), userID, taskID, req)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, model.TaskResponse{Data: *task})
}
