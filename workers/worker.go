package workers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go.uber.org/zap"
	"io"
	"model-hub/config"
	"model-hub/models"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"time"
)

type Worker struct {
	ID               WorkerId
	Model            config.Model
	Launched         bool
	Loaded           bool
	Busy             bool
	startTime        time.Time
	cmd              *exec.Cmd
	port             int
	mu               sync.Mutex
	failedWorkerChan chan WorkerId
	ctx              context.Context
	cancel           context.CancelFunc
	logger           *zap.Logger
}

func NewWorker(id WorkerId, model config.Model, port int, failedWorkerChan chan WorkerId, logger *zap.Logger) *Worker {
	ctx, cancel := context.WithCancel(context.Background())
	return &Worker{
		ID:               id,
		Model:            model,
		Loaded:           false,
		Busy:             false,
		Launched:         false,
		port:             port,
		failedWorkerChan: failedWorkerChan,
		ctx:              ctx,
		cancel:           cancel,
		logger:           logger,
	}
}

func (w *Worker) Start() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.cancel != nil {
		w.cancel()
	}
	w.ctx, w.cancel = context.WithCancel(context.Background())
	cmd := exec.Command("python3", "worker.py", string(w.ID), w.Model.Path, strconv.Itoa(w.port), w.Model.Handler)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stdout

	if err := cmd.Start(); err != nil {
		panic(fmt.Sprintf("failed to start worker %s: %v", w.ID, err))
	}
	w.Launched = true
	w.startTime = time.Now()

	go func() {
		err := cmd.Wait()
		if err != nil {
			timeString := w.ElapsedTimeString()

			w.logger.Error(fmt.Sprintf("Worker %s: command exited with error: %v, worked for %s", w.ID, err, timeString))
			w.failedWorkerChan <- w.ID
		}
	}()

	w.cmd = cmd
}

func (w *Worker) SetLoaded() {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.Loaded = true
}

func (w *Worker) IsLoaded() bool {
	w.mu.Lock()
	defer w.mu.Unlock()

	return w.Loaded
}

func (w *Worker) IsLaunched() bool {
	w.mu.Lock()
	defer w.mu.Unlock()

	return w.Launched
}

func (w *Worker) SetLaunched() {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.Launched = true
}

func (w *Worker) SetExited() {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.Launched = false
}

func (w *Worker) SetUnLoaded() {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.Loaded = false
}

func (w *Worker) SetBusy() {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.Busy = true
}

func (w *Worker) SetAvailable() {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.Busy = false
}

func (w *Worker) Predict(request models.PredictRequest) (response interface{}, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Marshal the request object
	reqBody, err := json.Marshal(request)
	if err != nil {
		return response, fmt.Errorf("worker %s: failed to marshal request: %v", w.ID, err)
	}

	// Create the POST request
	url := fmt.Sprintf("http://127.0.0.1:%d/predict", w.port)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return response, fmt.Errorf("worker %s: failed to create POST request: %v", w.ID, err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return response, fmt.Errorf("worker %s: failed to send POST request: %v", w.ID, err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return response, errors.New(string(respBody))
	}

	// Read and unmarshal the response
	if err != nil {
		return response, fmt.Errorf("worker %s: failed to read response body: %v", w.ID, err)
	}

	err = json.Unmarshal(respBody, &response)
	if err != nil {
		return response, fmt.Errorf("worker %s: failed to unmarshal response: %v", w.ID, err)
	}

	return response, nil
}
