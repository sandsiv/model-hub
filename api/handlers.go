package api

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"model-hub/models"
	"model-hub/workers"
	"net/http"
	"os"
)

type Handlers struct {
	manager *workers.WorkerManager
	logger  *zap.Logger
}

func NewHandlers(manager *workers.WorkerManager, logger *zap.Logger) *Handlers {
	return &Handlers{manager: manager, logger: logger}
}

func (h *Handlers) PredictHandler(c *gin.Context) {
	apiKey := os.Getenv("API_KEY")
	if apiKey != "" {
		clientAPIKey := c.GetHeader("X-API-KEY")
		if clientAPIKey != apiKey {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
	}

	var req models.PredictRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to decode request body"})
		return
	}
	modelString, ok := req.Params["model"].(string)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "model parameter is missing or has an invalid format"})
		return
	}
	priorityRaw, ok := req.Params["priority"].(float64)
	if !ok {
		priorityRaw = 1
	}
	priority := int(priorityRaw)
	model := models.ModelName(modelString)

	worker, err := h.manager.GetAvailableWorker(model, priority)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get available worker"})
		return
	}

	preds, err := worker.Predict(req)
	h.manager.SetWorkerAvailable(worker.ID)
	h.logComplete(ok, req, priority)
	if err != nil {
		c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, preds)
}

func (h *Handlers) logComplete(ok bool, req models.PredictRequest, priority int) {
	var info string
	metadata, ok := req.Params["metadata"].(string)
	if ok {
		info = fmt.Sprintf("%v. ", metadata)
	}
	info += fmt.Sprintf("(Priority: %d)", priority)
	h.logger.Info("Prediction complete. " + info)
}

func (h *Handlers) PingHandler(c *gin.Context) {
	c.Status(http.StatusOK)
}

func (h *Handlers) ModelReady(c *gin.Context) {
	var data struct {
		WorkerId workers.WorkerId `json:"worker_id"`
	}

	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to unmarshal request body"})
		return
	}

	h.manager.SetWorkerAvailable(data.WorkerId)
	c.Status(http.StatusOK)
}
