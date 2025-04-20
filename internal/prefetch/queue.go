package prefetch

import (
	"github.com/googlecloudplatform/gcsfuse/v2/internal/locker"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/logger"
)

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

// Len returns the number of item in the queue.
func (q *Queue) Len() uint64 {
	return q.length
}

type BlockQueue struct {
	q *Queue
}

func NewBlockQueue() *BlockQueue {
	return &BlockQueue{
		q: NewQueue(),
	}
}

func (bq *BlockQueue) IsEmpty() bool {
	return bq.q.IsEmpty()
}

func (bq *BlockQueue) Push(block *Block) {
	logger.Tracef("Pushed block %d\n", block.id)
	bq.q.Push(block)
}

func (bq *BlockQueue) Pop() *Block {
	val := bq.q.Pop()
	if val == nil {
		return nil
	}
	block, ok := val.(*Block)
	if !ok {
		panic("Block expected")
	}
	logger.Tracef("Popped block %d\n", block.id)
	return block
}

func (bq *BlockQueue) Peek() *Block {
	val := bq.q.Peek()
	if val == nil {
		return nil
	}
	block, ok := val.(*Block)
	if !ok {
		panic("Block expected")
	}
	return block
}
