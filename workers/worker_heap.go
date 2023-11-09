package workers

import "container/heap"

type WorkerRequest struct {
	worker     *Worker
	resultChan chan *Worker
	priority   int
	index      int
}

func NewWorkerRequest(priority int) *WorkerRequest {
	return &WorkerRequest{
		priority:   priority,
		resultChan: make(chan *Worker, 1), // Буферизированный канал, чтобы отправляющая горутина не блокировалась
	}
}

// WorkerQueue это куча (приоритетная очередь) запросов на воркера.
type WorkerQueue []*WorkerRequest

func (wq WorkerQueue) Len() int { return len(wq) }
func (wq WorkerQueue) Less(i, j int) bool {
	// Мы хотим Pop давать элемент с наивысшим приоритетом, поэтому используем > вместо <.
	return wq[i].priority > wq[j].priority
}
func (wq WorkerQueue) Swap(i, j int) {
	wq[i], wq[j] = wq[j], wq[i]
	wq[i].index = i
	wq[j].index = j
}

func (wq *WorkerQueue) Push(x interface{}) {
	n := len(*wq)
	item := x.(*WorkerRequest)
	item.index = n
	*wq = append(*wq, item)
}

func (wq *WorkerQueue) Pop() interface{} {
	old := *wq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil  // избегаем утечки памяти
	item.index = -1 // для безопасности
	*wq = old[0 : n-1]
	return item
}

func (wh *WorkerQueue) Remove(i int) interface{} {
	old := *wh
	item := old[i]
	*wh = append(old[:i], old[i+1:]...)
	heap.Init(wh)
	return item
}
