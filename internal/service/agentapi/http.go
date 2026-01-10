package agentapi

import (
	"github.com/dezemandje/aule/internal/domain"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// Handler handles HTTP requests for the agent API
type Handler struct {
	service *Service
}

// NewHandler creates a new agent API handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// GetTask handles GET /agent/v1/tasks/:task_id
func (h *Handler) GetTask(c *fiber.Ctx) error {
	taskIDStr := c.Params("task_id")
	taskUUID, err := uuid.Parse(taskIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(APIResponse{
			Success: false,
			Error:   "invalid task ID",
		})
	}

	taskID := domain.TaskID(taskUUID)
	resp, err := h.service.GetTaskDetails(c.Context(), taskID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(APIResponse{
			Success: false,
			Error:   err.Error(),
		})
	}

	return c.JSON(resp)
}

// StartTask handles POST /agent/v1/tasks/:task_id/start
func (h *Handler) StartTask(c *fiber.Ctx) error {
	taskIDStr := c.Params("task_id")
	taskUUID, err := uuid.Parse(taskIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(APIResponse{
			Success: false,
			Error:   "invalid task ID",
		})
	}

	// Get agent ID from auth context (set by middleware)
	agentID := c.Locals("agent_id")
	agentIDStr := ""
	if agentID != nil {
		agentIDStr = agentID.(string)
	}

	taskID := domain.TaskID(taskUUID)
	resp, err := h.service.StartTask(c.Context(), taskID, agentIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(APIResponse{
			Success: false,
			Error:   err.Error(),
		})
	}

	return c.JSON(resp)
}

// UpdateTask handles POST /agent/v1/tasks/:task_id/update
func (h *Handler) UpdateTask(c *fiber.Ctx) error {
	taskIDStr := c.Params("task_id")
	taskUUID, err := uuid.Parse(taskIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(APIResponse{
			Success: false,
			Error:   "invalid task ID",
		})
	}

	instanceIDStr := c.Query("instance_id")
	if instanceIDStr == "" {
		return c.Status(fiber.StatusBadRequest).JSON(APIResponse{
			Success: false,
			Error:   "instance_id query parameter required",
		})
	}

	instanceUUID, err := uuid.Parse(instanceIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(APIResponse{
			Success: false,
			Error:   "invalid instance ID",
		})
	}

	var req TaskUpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(APIResponse{
			Success: false,
			Error:   "invalid request body",
		})
	}

	taskID := domain.TaskID(taskUUID)
	instanceID := domain.AgentInstanceID(instanceUUID)

	err = h.service.UpdateTask(c.Context(), taskID, instanceID, &req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(APIResponse{
			Success: false,
			Error:   err.Error(),
		})
	}

	return c.JSON(APIResponse{Success: true})
}

// CompleteTask handles POST /agent/v1/tasks/:task_id/complete
func (h *Handler) CompleteTask(c *fiber.Ctx) error {
	taskIDStr := c.Params("task_id")
	taskUUID, err := uuid.Parse(taskIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(APIResponse{
			Success: false,
			Error:   "invalid task ID",
		})
	}

	instanceIDStr := c.Query("instance_id")
	if instanceIDStr == "" {
		return c.Status(fiber.StatusBadRequest).JSON(APIResponse{
			Success: false,
			Error:   "instance_id query parameter required",
		})
	}

	instanceUUID, err := uuid.Parse(instanceIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(APIResponse{
			Success: false,
			Error:   "invalid instance ID",
		})
	}

	var req TaskCompleteRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(APIResponse{
			Success: false,
			Error:   "invalid request body",
		})
	}

	taskID := domain.TaskID(taskUUID)
	instanceID := domain.AgentInstanceID(instanceUUID)

	err = h.service.CompleteTask(c.Context(), taskID, instanceID, &req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(APIResponse{
			Success: false,
			Error:   err.Error(),
		})
	}

	return c.JSON(APIResponse{Success: true, Message: "task completed"})
}

// FailTask handles POST /agent/v1/tasks/:task_id/fail
func (h *Handler) FailTask(c *fiber.Ctx) error {
	taskIDStr := c.Params("task_id")
	taskUUID, err := uuid.Parse(taskIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(APIResponse{
			Success: false,
			Error:   "invalid task ID",
		})
	}

	instanceIDStr := c.Query("instance_id")
	if instanceIDStr == "" {
		return c.Status(fiber.StatusBadRequest).JSON(APIResponse{
			Success: false,
			Error:   "instance_id query parameter required",
		})
	}

	instanceUUID, err := uuid.Parse(instanceIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(APIResponse{
			Success: false,
			Error:   "invalid instance ID",
		})
	}

	var req TaskFailRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(APIResponse{
			Success: false,
			Error:   "invalid request body",
		})
	}

	taskID := domain.TaskID(taskUUID)
	instanceID := domain.AgentInstanceID(instanceUUID)

	err = h.service.FailTask(c.Context(), taskID, instanceID, &req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(APIResponse{
			Success: false,
			Error:   err.Error(),
		})
	}

	return c.JSON(APIResponse{Success: true, Message: "task marked as failed"})
}
