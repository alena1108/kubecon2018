package util

import (
	"time"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

// TaskQueue manages a work queue through an independent worker that
// invokes the given sync function for every work item inserted.
type TaskQueue struct {
	// queue is the work queue the worker polls
	queue *workqueue.Type
	// sync is called for each item in the queue
	sync func(string)
	// workerDone is closed when the worker exits
	workerDone chan struct{}
}

func (t *TaskQueue) Run(period time.Duration, stopCh <-chan struct{}) {
	wait.Until(t.worker, period, stopCh)
}

// Enqueue enqueues ns/name of the given api object in the task queue.
func (t *TaskQueue) Enqueue(obj interface{}) {
	if key, ok := obj.(string); ok {
		t.queue.Add(key)
	} else {
		key, err := cache.MetaNamespaceKeyFunc(obj)
		if err != nil {
			logrus.Infof("could not get key for object %+v: %v", obj, err)
			return
		}
		t.queue.Add(key)
	}
}

func (t *TaskQueue) Requeue(key string, err error) {
	logrus.Debugf("requeuing %v, err %v", key, err)
	t.queue.Add(key)
}

// worker processes work in the queue through sync.
func (t *TaskQueue) worker() {
	for {
		key, quit := t.queue.Get()
		if quit {
			close(t.workerDone)
			return
		}
		logrus.Debugf("syncing %v", key)
		t.sync(key.(string))
		t.queue.Done(key)
	}
}

// Shutdown shuts down the work queue and waits for the worker to ACK
func (t *TaskQueue) Shutdown() {
	t.queue.ShutDown()
	<-t.workerDone
}

// NewTaskQueue creates a new task queue with the given sync function.
// The sync function is called for every element inserted into the queue.
func NewTaskQueue(syncFn func(string)) *TaskQueue {
	return &TaskQueue{
		queue:      workqueue.New(),
		sync:       syncFn,
		workerDone: make(chan struct{}),
	}
}
