package tsync

import (
	"github.com/trivago/tgo/ttesting"
	"sync/atomic"
	"testing"
	"time"
)

func TestFunctionality(t *testing.T) {
	expect := ttesting.NewExpect(t)
	q := NewQueue(1)

	expect.NonBlocking(time.Second, func() { expect.NoError(q.Push(1)) })
	expect.False(q.IsEmpty())

	v := q.Pop()
	expect.Equal(1, v)
	expect.True(q.IsEmpty())
	expect.False(q.IsDrained())

	expect.NonBlocking(time.Second, func() { expect.NoError(q.Push(2)) })
	q.Close()

	v = q.Pop()
	expect.Equal(2, v)
	expect.True(q.IsEmpty())
	expect.True(q.IsDrained())

	err := q.Push(3)
	expect.OfType(LockedError{}, err)
}

func TestConcurrency(t *testing.T) {
	expect := ttesting.NewExpect(t)
	q := NewQueue(100)

	writer := WaitGroup{}
	reader := WaitGroup{}
	numSamples := 1000

	results := make([]*uint64, 20)
	writes := new(uint32)

	// Start writer
	for i := 0; i < len(results); i++ {
		results[i] = new(uint64)
		idx := i
		writer.Add(1)
		go func() {
			defer writer.Done()
			for m := 0; m < numSamples; m++ {
				time.Sleep(time.Microsecond * 100)
				expect.NoError(q.Push(idx))
				atomic.AddUint32(writes, 1)
			}
		}()
	}

	// Start reader
	for i := 0; i < 10; i++ {
		reader.Add(1)
		go func() {
			defer reader.Done()
			for !q.IsDrained() {
				value := q.Pop()
				if value != nil {
					idx, _ := value.(int)
					atomic.AddUint64(results[idx], 1)
				}
			}
		}()
	}

	// Give them some time
	writer.WaitFor(time.Second)
	expect.Equal(int32(0), writer.counter)
	expect.Equal(len(results)*numSamples, int(atomic.LoadUint32(writes)))

	q.Close()
	reader.WaitFor(time.Second)
	expect.Equal(int32(0), reader.counter)

	// Check results
	for i := 0; i < len(results); i++ {
		expect.Equal(uint64(numSamples), atomic.LoadUint64(results[i]))
	}
}

func BenchmarkQueuePush(b *testing.B) {
	for i := 0; i < b.N; i++ {
		q := NewQueue(100000)
		for c := 0; c < 100000; c++ {
			q.Push(123)
		}
	}
}

func BenchmarkChannelPush(b *testing.B) {
	for i := 0; i < b.N; i++ {
		q := make(chan interface{}, 100000)
		for c := 0; c < 100000; c++ {
			q <- 123
		}
	}
}
