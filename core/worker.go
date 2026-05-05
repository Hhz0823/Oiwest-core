package core

import (
	"context"
	"log"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

type Task func(ctx context.Context) error

type TaskResult struct {
	Err    error
	TaskID int64
}

type WorkerPool struct {
	tasks      chan Task
	results    chan TaskResult
	numWorkers int
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	active     int32
	submitted  int64
	completed  int64
	mu         sync.RWMutex
	stopped    int32
}

type WorkerPoolConfig struct {
	NumWorkers  int           `json:"numWorkers"`
	QueueSize   int           `json:"queueSize"`
	MaxRetries  int           `json:"maxRetries"`
	IdleTimeout time.Duration `json:"idleTimeout"`
}

func DefaultWorkerPoolConfig() *WorkerPoolConfig {
	numCPU := runtime.NumCPU()
	return &WorkerPoolConfig{
		NumWorkers:  numCPU * 2,
		QueueSize:   numCPU * 100,
		MaxRetries:  3,
		IdleTimeout: 30 * time.Second,
	}
}

func NewWorkerPool(config *WorkerPoolConfig) *WorkerPool {
	if config == nil {
		config = DefaultWorkerPoolConfig()
	}
	ctx, cancel := context.WithCancel(context.Background())
	pool := &WorkerPool{
		tasks:      make(chan Task, config.QueueSize),
		results:    make(chan TaskResult, config.QueueSize),
		numWorkers: config.NumWorkers,
		ctx:        ctx,
		cancel:     cancel,
	}
	pool.Start()
	return pool
}

func (p *WorkerPool) Start() {
	p.mu.Lock()
	defer p.mu.Unlock()
	for i := 0; i < p.numWorkers; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}
	log.Printf("[WorkerPool] Started %d workers", p.numWorkers)
}

func (p *WorkerPool) worker(id int) {
	defer p.wg.Done()
	for {
		select {
		case task, ok := <-p.tasks:
			if !ok {
				return
			}
			atomic.AddInt32(&p.active, 1)
			taskID := atomic.AddInt64(&p.submitted, 1)
			err := p.executeTask(task, taskID)
			atomic.AddInt32(&p.active, -1)
			atomic.AddInt64(&p.completed, 1)
			select {
			case p.results <- TaskResult{Err: err, TaskID: taskID}:
			default:
			}
		case <-p.ctx.Done():
			return
		}
	}
}

func (p *WorkerPool) executeTask(task Task, taskID int64) error {
	if task == nil {
		return nil
	}
	return task(p.ctx)
}

func (p *WorkerPool) Submit(task Task) error {
	if atomic.LoadInt32(&p.stopped) != 0 {
		return nil
	}
	select {
	case p.tasks <- task:
		return nil
	case <-p.ctx.Done():
		return p.ctx.Err()
	}
}

func (p *WorkerPool) SubmitBatch(tasks []Task) []error {
	errs := make([]error, len(tasks))
	for i, task := range tasks {
		if err := p.Submit(task); err != nil {
			errs[i] = err
		}
	}
	return errs
}

func (p *WorkerPool) Results() <-chan TaskResult {
	return p.results
}

func (p *WorkerPool) Active() int {
	return int(atomic.LoadInt32(&p.active))
}

func (p *WorkerPool) Submitted() int64 {
	return atomic.LoadInt64(&p.submitted)
}

func (p *WorkerPool) Completed() int64 {
	return atomic.LoadInt64(&p.completed)
}

func (p *WorkerPool) NumWorkers() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.numWorkers
}

func (p *WorkerPool) Pending() int {
	return len(p.tasks)
}

func (p *WorkerPool) Resize(numWorkers int) {
	if numWorkers <= 0 {
		return
	}
	p.mu.Lock()
	oldNum := p.numWorkers
	p.numWorkers = numWorkers
	p.mu.Unlock()

	if numWorkers > oldNum {
		for i := oldNum; i < numWorkers; i++ {
			p.wg.Add(1)
			go p.worker(i)
		}
	}
}

func (p *WorkerPool) Stop() {
	if atomic.CompareAndSwapInt32(&p.stopped, 0, 1) {
		p.cancel()
	}
}

func (p *WorkerPool) Shutdown() {
	p.Stop()
	p.wg.Wait()
}

func (p *WorkerPool) ShutdownAndWait(timeout time.Duration) error {
	done := make(chan struct{})
	go func() {
		p.Shutdown()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-time.After(timeout):
		return nil
	}
}

type TaskScheduler struct {
	pool       *WorkerPool
	pending    []Task
	priorities map[string]int
	mu         sync.Mutex
}

func NewTaskScheduler(pool *WorkerPool) *TaskScheduler {
	return &TaskScheduler{
		pool:       pool,
		priorities: make(map[string]int),
	}
}

func (s *TaskScheduler) Schedule(task Task, priority int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if priority > 0 {
		highTasks := make([]Task, 0, len(s.pending)+1)
		highTasks = append(highTasks, task)
		highTasks = append(highTasks, s.pending...)
		s.pending = highTasks
	} else {
		s.pending = append(s.pending, task)
	}
}

func (s *TaskScheduler) Flush() {
	s.mu.Lock()
	tasks := make([]Task, len(s.pending))
	copy(tasks, s.pending)
	s.pending = s.pending[:0]
	s.mu.Unlock()

	for _, task := range tasks {
		s.pool.Submit(task)
	}
}

type ParallelExecutor struct {
	maxConcurrency int
	semaphore      chan struct{}
	wg             sync.WaitGroup
}

func NewParallelExecutor(maxConcurrency int) *ParallelExecutor {
	if maxConcurrency <= 0 {
		maxConcurrency = runtime.NumCPU() * 2
	}
	return &ParallelExecutor{
		maxConcurrency: maxConcurrency,
		semaphore:      make(chan struct{}, maxConcurrency),
	}
}

func (e *ParallelExecutor) Execute(task func()) {
	e.semaphore <- struct{}{}
	e.wg.Add(1)
	go func() {
		defer e.wg.Done()
		defer func() { <-e.semaphore }()
		task()
	}()
}

func (e *ParallelExecutor) Wait() {
	e.wg.Wait()
}

func (e *ParallelExecutor) ExecuteBatch(tasks []func()) {
	for _, task := range tasks {
		e.Execute(task)
	}
	e.Wait()
}
