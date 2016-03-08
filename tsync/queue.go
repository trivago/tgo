package tsync

import (
	"sync/atomic"
)

// Queue implements a multi-producer, multi-consumer, lockfree queue.
// Push is waitfree as long as the queue is not full.
// Pop is waitfree as long as there are items in the queue.
type Queue struct {
	items  []interface{}
	read   queueAccess
	write  queueAccess
	locked *int32
}

// NewQueue creates a new queue with medium spinning priority
func NewQueue(capacity uint32) Queue {
	return NewQueueWithPriority(capacity, SpinPriorityMedium)
}

// NewQueueWithPriority allows to set the spinning priority of the queue to
// be created.
func NewQueueWithPriority(capacity uint32, priority SpinPriority) Queue {
	return Queue{
		items:  make([]interface{}, capacity),
		read:   newQueueAccess(capacity, priority),
		write:  newQueueAccess(capacity, priority),
		locked: new(int32),
	}
}

// Close blocks the queue from write access. It also allows Pop() to return
// false as a second return value
func (q *Queue) Close() {
	atomic.StoreInt32(q.locked, 1)
}

// Push adds an item to the queue. This call may block if the queue is full.
// An error is returned when the queue is locked.
func (q *Queue) Push(item interface{}) error {
	if atomic.LoadInt32(q.locked) == 1 {
		return LockedError{"Queue is locked"}
	}
	slot, ticket := q.write.getTicket()
	q.read.waitIfOverlapping(ticket)

	q.items[slot] = item
	q.write.ticketDone(ticket)
	return nil
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

// Pop removes an item from the queue. This call may block if the queue is
// empty. If the queue is drained Pop() will not block and return nil.
func (q *Queue) Pop() interface{} {
	slot, ticket := q.read.getTicket()
	q.write.waitUntilReady(ticket, q.IsDrained)
	if q.IsDrained() {
		atomic.StoreUint64(q.read.next, atomic.LoadUint64(q.write.next))
		return nil
	}

	item := q.items[slot]
	q.read.ticketDone(ticket)
	return item
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
	capacity   uint64
	spin       Spinner
}

func newQueueAccess(capacity uint32, priority SpinPriority) queueAccess {
	return queueAccess{
		processing: new(uint64),
		next:       new(uint64),
		capacity:   uint64(capacity),
		spin:       NewSpinner(priority),
	}
}

func (a queueAccess) getTicket() (slot uint32, ticket uint64) {
	ticket = atomic.AddUint64(a.next, 1) - 1
	a.waitIfOverlapping(ticket)
	return uint32(ticket % a.capacity), ticket
}

func (a queueAccess) ticketDone(ticket uint64) {
	for ticket != atomic.LoadUint64(a.processing) {
		a.spin.Yield()
	}
	atomic.AddUint64(a.processing, 1)
}

func (a queueAccess) waitUntilReady(ticket uint64, abort func() bool) {
	for ticket >= atomic.LoadUint64(a.processing) && !abort() {
		a.spin.Yield()
	}
}

func (a queueAccess) waitIfOverlapping(ticket uint64) {
	for ticket-atomic.LoadUint64(a.processing) >= a.capacity {
		a.spin.Yield()
	}
}

func (a queueAccess) isTicketInUse(ticket uint64) bool {
	return ticket < atomic.LoadUint64(a.next)
}
