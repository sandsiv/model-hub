package workers

import (
	"container/heap"
	"fmt"
	"go.uber.org/zap"
	"model-hub/config"
	"model-hub/helper"
	"model-hub/models"
	"sync"
	"time"
)

type WorkerId string

type WorkerManager struct {
	workers             map[WorkerId]*Worker                     // Existing workers
	failedWorkerChan    chan WorkerId                            // Channel for failed workers
	workerAvailableChan map[models.ModelName]chan WorkerId       // Channel for notification about worker ready
	modelNames          []models.ModelName                       // Models list
	workerRequestChan   map[models.ModelName]chan *WorkerRequest // WorkerRequest channale by models
	workerQueues        map[models.ModelName]*WorkerQueue        // WorkerQueue heap by model
	mu                  sync.Mutex
	logger              *zap.Logger
}

func NewWorkerManager(cfg *config.Config, logger *zap.Logger) *WorkerManager {
	workers := make(map[WorkerId]*Worker)
	workerChan := make(map[models.ModelName]chan *Worker)
	workerQueues := make(map[models.ModelName]*WorkerQueue)
	workerRequestChan := make(map[models.ModelName]chan *WorkerRequest)
	var modelNames []models.ModelName
	port := 7777
	workerAvailableChan := make(map[models.ModelName]chan WorkerId)
	failedWorkerChan := make(chan WorkerId)
	for _, model := range cfg.Models {
		modelNames = append(modelNames, model.Name)
		workerChan[model.Name] = make(chan *Worker, model.Workers)
		for i := 1; i <= model.Workers; i++ {
			port += 1
			workerID := WorkerId(fmt.Sprintf("%s-%d", model.Name, i))
			worker := NewWorker(workerID, model, port, failedWorkerChan, logger)
			workers[workerID] = worker
		}
		workerRequestChan[model.Name] = make(chan *WorkerRequest)
		workerQueues[model.Name] = new(WorkerQueue)
		workerAvailableChan[model.Name] = make(chan WorkerId, model.Workers)
		heap.Init(workerQueues[model.Name])
	}
	return &WorkerManager{
		workers:             workers,
		workerAvailableChan: workerAvailableChan,
		failedWorkerChan:    failedWorkerChan,
		modelNames:          modelNames,
		logger:              logger,
		workerRequestChan:   workerRequestChan,
		workerQueues:        workerQueues,
	}
}

func (wm *WorkerManager) removeWorkerFromChannel(worker *Worker) {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	// Теперь обрабатываем очередь запросов на воркеров, удаляем воркера из всех запросов
	workerQueue := wm.workerQueues[worker.Model.Name]
	newQueue := new(WorkerQueue)
	heap.Init(newQueue)

	for _, request := range *workerQueue {
		if request.worker == nil || request.worker.ID != worker.ID {
			heap.Push(newQueue, request)
		}
	}

	wm.workerQueues[worker.Model.Name] = newQueue
}

func (wm *WorkerManager) handleFailedWorker() {
	for {
		failedWorkerID := <-wm.failedWorkerChan
		worker, ok := wm.workers[failedWorkerID]
		if ok {
			go func() {
				worker.SetUnLoaded()
				worker.SetExited()
				wm.removeWorkerFromChannel(worker)
				wm.logger.Info(fmt.Sprintf("Worker %s: Waiting 5 seconds before restarting", worker.ID))
				time.Sleep(5 * time.Second)
				worker.Start()
			}()
		}
	}
}

func (wm *WorkerManager) Initialize() {
	for _, modelName := range wm.modelNames {
		go wm.processWorkerRequests(modelName)
	}
	go wm.handleFailedWorker()
	go wm.logResourceUsage()
	loadingStrategy := helper.GetEnv("WORKERS_LOADING_STRATEGY", "parallel")
	if loadingStrategy == "sequential" {
		wm.startWorkersSequentially()
	} else {
		wm.startWorkersParallel()
	}
}
func (wm *WorkerManager) processWorkerRequests(modelName models.ModelName) {
	for {
		select {
		case request := <-wm.workerRequestChan[modelName]:
			// Add Request to heap
			wm.mu.Lock()
			heap.Push(wm.workerQueues[modelName], request)
			wm.mu.Unlock()

		case workerId := <-wm.workerAvailableChan[modelName]:
			wm.mu.Lock()
			if wm.workerQueues[modelName].Len() == 0 {
				wm.workerAvailableChan[modelName] <- workerId
				wm.mu.Unlock()
				continue
			}

			// Fetch prioritized request from heap
			nextRequest := heap.Pop(wm.workerQueues[modelName]).(*WorkerRequest)
			wm.mu.Unlock()

			worker, exists := wm.workers[workerId]
			if exists {
				nextRequest.worker = worker
				worker.SetBusy()
				nextRequest.resultChan <- worker
			} else {
				wm.logger.Error("Worker not found", zap.String("workerId", string(workerId)))
			}
		}
	}
}

func (wm *WorkerManager) startWorkersSequentially() {
	for _, worker := range wm.workers {
		worker.Start()
		for !worker.IsLoaded() {
			time.Sleep(1 * time.Second)
		}
	}
}

func (wm *WorkerManager) startWorkersParallel() {
	for _, worker := range wm.workers {
		worker.Start()
	}
}

func (wm *WorkerManager) GetAvailableWorker(modelName models.ModelName, priority int) (*Worker, error) {
	if _, ok := wm.workerRequestChan[modelName]; !ok {
		return nil, fmt.Errorf("no worker channel for the requested model: %s", modelName)
	}
	request := NewWorkerRequest(priority)
	wm.workerRequestChan[modelName] <- request
	worker := <-request.resultChan

	return worker, nil
}

func (wm *WorkerManager) SetWorkerAvailable(workerID WorkerId) {
	worker, ok := wm.workers[workerID]
	if ok {
		worker.SetAvailable()

		wm.workerAvailableChan[worker.Model.Name] <- workerID
	}
}
