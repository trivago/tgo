// Copyright 2015-2016 trivago GmbH
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tsync

import (
	"github.com/trivago/tgo/ttesting"
	"runtime"
	"sync"
	"testing"
)

func TestStackFunctionality(t *testing.T) {
	expect := ttesting.NewExpect(t)
	s := NewStack(1)

	v, err := s.Pop()
	expect.Nil(v)

	s.Push(1)
	expect.Equal(1, len(s.data))

	s.Push(2)
	expect.Equal(2, len(s.data))

	v, err = s.Pop()
	expect.NoError(err)
	expect.Equal(2, v.(int))

	v, err = s.Pop()
	expect.NoError(err)
	expect.Equal(1, v.(int))

	v, err = s.Pop()
	expect.Nil(v)
}

func TestStackConcurrency(t *testing.T) {
	expect := ttesting.NewExpect(t)
	s := NewStack(1)
	start := sync.WaitGroup{}
	end := sync.WaitGroup{}
	start.Add(1)

	go func() {
		end.Add(1)
		defer end.Done()
		start.Wait()

		for i := 0; i < 10000; i++ {
			s.Push(i)
			runtime.Gosched()
		}
	}()

	// Start reader threads
	for r := 0; r < 10; r++ {
		go func() {
			end.Add(1)
			defer end.Done()
			start.Wait()

			lastValue := -1
			errCount := 0
			for i := 0; i < 1000; i++ {
				v, err := s.Pop()
				if err == nil {
					expect.Greater(v.(int), lastValue)
					lastValue = v.(int)
				} else {
					errCount++
				}
				runtime.Gosched()
			}

			expect.Less(errCount, 1000)
		}()
	}

	start.Done()
	end.Wait()
}
