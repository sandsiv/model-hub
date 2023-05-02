package api

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"model-hub/models"
	"model-hub/workers"
	"net/http"
)

type Handlers struct {
	manager *workers.WorkerManager
	logger  *zap.Logger
}

func NewHandlers(manager *workers.WorkerManager, logger *zap.Logger) *Handlers {
	return &Handlers{manager: manager, logger: logger}
}

func (h *Handlers) PredictHandler(c *gin.Context) {
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
	model := models.ModelName(modelString)

	worker, err := h.manager.GetAvailableWorker(model)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get available worker"})
		return
	}

	preds, err := worker.Predict(req)
	h.manager.SetWorkerAvailable(worker.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, preds)
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
