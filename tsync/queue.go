package tsync

import (
	"sync/atomic"
)

// Queue implements a multi-producer, multi-consumer, lockfree queue.
// Push is waitfree as long as the queue is not full.
// Pop is waitfree as long as there are items in the queue.
type Queue struct {
	write    queueAccess
	read     queueAccess
	capacity uint64
	locked   *int32
	priority SpinPriority
	items    []interface{}
}

// NewQueue creates a new queue with medium spinning priority
func NewQueue(capacity uint32) Queue {
	return NewQueueWithPriority(capacity, SpinPriorityMedium)
}

// NewQueueWithPriority allows to set the spinning priority of the queue to
// be created.
func NewQueueWithPriority(capacity uint32, priority SpinPriority) Queue {
	return Queue{
		items:    make([]interface{}, capacity),
		read:     newQueueAccess(),
		write:    newQueueAccess(),
		locked:   new(int32),
		capacity: uint64(capacity),
		priority: priority,
	}
}

// Push adds an item to the queue. This call may block if the queue is full.
// An error is returned when the queue is locked.
func (q *Queue) Push(item interface{}) error {
	if atomic.LoadInt32(q.locked) == 1 {
		return LockedError{"Queue is locked"} // ### return, closed ###
	}

	// Get ticket and slot
	ticket := atomic.AddUint64(q.write.next, 1) - 1
	slot := ticket % q.capacity
	spin := NewSpinner(q.priority)

	// Wait for pending reads on slot
	for ticket-atomic.LoadUint64(q.read.processing) >= q.capacity {
		spin.Yield()
	}

	q.items[slot] = item

	// Wait for previous writers to finish writing
	for ticket != atomic.LoadUint64(q.write.processing) {
		spin.Yield()
	}
	atomic.AddUint64(q.write.processing, 1)
	return nil
}

// Pop removes an item from the queue. This call may block if the queue is
// empty. If the queue is drained Pop() will not block and return nil.
func (q *Queue) Pop() interface{} {
	// Drained?
	if atomic.LoadInt32(q.locked) == 1 &&
		atomic.LoadUint64(q.write.processing) == atomic.LoadUint64(q.read.processing) {
		return nil // ### return, closed and no items ###
	}

	// Get ticket andd slot
	ticket := atomic.AddUint64(q.read.next, 1) - 1
	slot := ticket % q.capacity
	spin := NewSpinner(q.priority)

	// Wait for slot to be written to
	for ticket >= atomic.LoadUint64(q.write.processing) {
		spin.Yield()
		// Drained?
		if atomic.LoadInt32(q.locked) == 1 &&
			atomic.LoadUint64(q.write.processing) == atomic.LoadUint64(q.read.processing) {
			return nil // ### return, closed while spinning ###
		}
	}

	item := q.items[slot]

	// Wait for other reads to finish
	for ticket != atomic.LoadUint64(q.read.processing) {
		spin.Yield()
	}
	atomic.AddUint64(q.read.processing, 1)
	return item
}

// Close blocks the queue from write access. It also allows Pop() to return
// false as a second return value
func (q *Queue) Close() {
	atomic.StoreInt32(q.locked, 1)
}

// Reopen unblocks the queue to allow write access again.
func (q *Queue) Reopen() {
	atomic.StoreInt32(q.locked, 0)
}

// IsClosed returns true if Close() has been called.
func (q *Queue) IsClosed() bool {
	return atomic.LoadInt32(q.locked) == 1
}

// IsEmpty returns true if there is no item in the queue to be processed.
// Please note that this state is extremely volatile unless IsClosed
// returned true.
func (q *Queue) IsEmpty() bool {
	return atomic.LoadUint64(q.write.processing) == atomic.LoadUint64(q.read.processing)
}

// IsDrained combines IsClosed and IsEmpty.
func (q *Queue) IsDrained() bool {
	return q.IsClosed() && q.IsEmpty()
}

// Queue access encapsulates the two-index-access pattern for this queue.
// If one or both indices overflow there will be errors. This happens after
// 18 * 10^18 writes aka never if you are not doing more than 10^11 writes
// per second (overflow after ~694 days).
// Just to put this into perspective - an Intel Core i7 5960X at 3.5 Ghz can
// do 336 * 10^9 ops per second.
type queueAccess struct {
	processing *uint64
	next       *uint64
}

func newQueueAccess() queueAccess {
	return queueAccess{
		processing: new(uint64),
		next:       new(uint64),
	}
}