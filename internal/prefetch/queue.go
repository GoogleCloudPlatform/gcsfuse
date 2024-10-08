package prefetch

import "github.com/googlecloudplatform/gcsfuse/v2/internal/locker"

type entry struct {
	value interface{}
	next  *entry
}

type Queue struct {
	start, end *entry
	length     uint64
	mu         locker.RWLocker
}

func NewQueue() *Queue {
	return &Queue{
		nil,
		nil,
		0,
		locker.NewRW("Queue", func() {}),
	}
}

// IsEmpty tells queue is empty or not.
func (q *Queue) IsEmpty() bool {
	return q.length == 0
}

// Peek returns the first item in the queue without remove it. Returns nil if queue is empty.
func (q *Queue) Peek() interface{} {
	q.mu.RLock()
	defer q.mu.RUnlock()

	if q.length == 0 {
		return nil
	}
	return q.start.value
}

// Push puts an item on the end of the queue.
func (q *Queue) Push(value interface{}) {
	n := &entry{value, nil}
	if q.length == 0 {
		q.start = n
		q.end = n
	} else {
		q.end.next = n
		q.end = n
	}
	q.length++
}

// Pop removes and returns the front item from the queue.
func (q *Queue) Pop() interface{} {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.length == 0 {
		return nil
	}

	n := q.start
	if q.length == 1 {
		q.start = nil
		q.end = nil
	} else {
		q.start = q.start.next
	}
	q.length--
	return n.value
}
