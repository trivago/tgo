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

package tcontainer

import (
	"github.com/trivago/tgo/ttesting"
	"runtime"
	"testing"
	"time"
)

func TestBytePool(t *testing.T) {
	expect := ttesting.NewExpect(t)
	pool := NewBytePool()

	tinyMin := pool.Get(1)
	expect.Equal(64, cap(tinyMin))
	expect.Equal(1, len(tinyMin))

	tinyMax := pool.Get(960)
	expect.Equal(960, cap(tinyMax))
	expect.Equal(960, len(tinyMax))

	smallMin := pool.Get(961)
	expect.Equal(1024, cap(smallMin))
	expect.Equal(961, len(smallMin))

	smallMax := pool.Get(1024 * 9)
	expect.Equal(1024*9, cap(smallMax))
	expect.Equal(1024*9, len(smallMax))

	mediumMin := pool.Get(1024*9 + 1)
	expect.Equal(1024*10, cap(mediumMin))
	expect.Equal(1024*9+1, len(mediumMin))

	mediumMax := pool.Get(1024 * 90)
	expect.Equal(1024*90, cap(mediumMax))
	expect.Equal(1024*90, len(mediumMax))

	largeMin := pool.Get(1024*90 + 1)
	expect.Equal(1024*100, cap(largeMin))
	expect.Equal(1024*90+1, len(largeMin))

	largeMax := pool.Get(1024 * 1000)
	expect.Equal(1024*1000, cap(largeMax))
	expect.Equal(1024*1000, len(largeMax))

	huge := pool.Get(1024*1000 + 1)
	expect.Equal(1024*1000+1, cap(huge))
	expect.Equal(1024*1000+1, len(huge))
}

func allocateWaste(pool *BytePool, expect ttesting.Expect) {
	data := pool.Get(32)
	for i := 0; i < 32; i++ {
		data[i] = byte(i)
	}

	expect.Equal(0, len(pool.tiny.slabs[0]))
}

func TestBytePoolRecycle(t *testing.T) {
	expect := ttesting.NewExpect(t)
	pool := NewBytePoolWithSize(1, 1, 1, 1)

	expect.Nil(pool.tiny.slabs[0])
	allocateWaste(&pool, expect)

	expect.NonBlocking(time.Second, func() {
		for len(pool.tiny.slabs[0]) == 0 {
			runtime.Gosched()
			runtime.GC()
		}
	})

	expect.Equal(1, len(pool.tiny.slabs[0]))

	data := pool.Get(32)
	for i := 0; i < 32; i++ {
		expect.Equal(byte(i), data[i])
	}
}
