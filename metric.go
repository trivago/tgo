// Copyright 2015 trivago GmbH
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

package tgo

import (
	"encoding/json"
	"fmt"
	"github.com/trivago/tgo/tcontainer"
	"github.com/trivago/tgo/tmath"
	"math"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	// MetricProcessStart is the metric name storing the time when this process
	// has been started.
	MetricProcessStart = "ProcessStart"
	// MetricGoRoutines is the metric name storing the number of active go
	// routines.
	MetricGoRoutines = "GoRoutines"
	// MetricGoVersion holds the go version as Major*10000+Minor*100+Patch
	MetricGoVersion = "GoVersion"
	// MetricMemoryAllocated holds the currently active memory in bytes
	MetricMemoryAllocated = "GoMemoryAllocated"
	// MetricMemoryNumObjects holds the total number of allocated heap objects
	MetricMemoryNumObjects = "GoMemoryNumObjects"
	// MetricMemoryGCEnabled holds 1 or 0 depending on the state of garbage collection
	MetricMemoryGCEnabled = "GoMemoryGCEnabled"
)

// ProcessStartTime stores the time this process has started.
// This value is also stored in the metric MetricProcessStart
var ProcessStartTime time.Time

func init() {
	ProcessStartTime = time.Now()
}

// Metrics is the container struct for runtime metrics that can be used with
// the metrics server.
type Metrics struct {
	store      map[string]*int64
	rates      map[string]*rate
	ticker     *time.Ticker
	storeGuard *sync.RWMutex
	rateGuard  *sync.RWMutex
}

type rate struct {
	metric     string
	samples    tcontainer.Int64Slice
	lastSample int64
	value      int64
	numMedians int
	index      int
}

// Metric allows any part of gollum to store and/or modify metric values by
// name.
var Metric = (*Metrics)(nil)

// EnableGlobalMetrics initializes the Metric global variable (if it is not nil)
// This function is not threadsafe and should be called once directly after the
// process started.
func EnableGlobalMetrics() {
	if Metric == nil {
		Metric = NewMetrics()
	}
}

// NewMetrics creates a new metrics container.
// To initialize the global Metrics variable use EnableGlobalMetrics.
func NewMetrics() *Metrics {
	metrics := &Metrics{
		store:      make(map[string]*int64),
		rates:      make(map[string]*rate),
		ticker:     time.NewTicker(time.Second),
		storeGuard: new(sync.RWMutex),
		rateGuard:  new(sync.RWMutex),
	}

	metrics.New(MetricProcessStart)
	metrics.New(MetricGoRoutines)
	metrics.New(MetricGoVersion)
	metrics.New(MetricMemoryAllocated)
	metrics.New(MetricMemoryNumObjects)
	metrics.New(MetricMemoryGCEnabled)
	metrics.Set(MetricProcessStart, ProcessStartTime.Unix())

	version := runtime.Version()
	if version[0] == 'g' && version[1] == 'o' {
		parts := strings.Split(version[2:], ".")
		numericVersion := make([]uint64, tmath.MaxI(3, len(parts)))
		for i, p := range parts {
			numericVersion[i], _ = strconv.ParseUint(p, 10, 64)
		}

		metrics.SetI(MetricGoVersion, int(numericVersion[0]*10000+numericVersion[1]*100+numericVersion[2]))
	}

	go metrics.rateSampler()
	return metrics
}

// Close stops the internal go routines used for e.g. sampling
func (met *Metrics) Close() {
	met.ticker.Stop()
}

// New creates a new metric under the given name with a value of 0
func (met *Metrics) New(name string) {
	met.storeGuard.Lock()
	defer met.storeGuard.Unlock()
	if _, exists := met.store[name]; !exists {
		met.store[name] = new(int64)
	}
}

// NewRate creates a new rate. Rates are based on another metric and sample
// this given base metric every second. When numSamples have been stored, old
// samples are overriden (oldest first).
// Retrieving samples via GetRate will calculate the median of a set of means.
// numMedianSamples defines how many values will be used for mean calculation.
// A value of 0 will calculate the mean value of all samples. A value of 1 or
// a value >= numSamples will build a median over all samples. Any other
// value will divide the stored samples into the given number of groups and
// build a median over the mean of all these groups.
func (met *Metrics) NewRate(baseMetric string, name string, numSamples uint8, numMedianSamples uint8) error {
	met.storeGuard.RLock()
	if _, exists := met.store[baseMetric]; !exists {
		met.storeGuard.RUnlock()
		return fmt.Errorf("Metric %s is not registered", baseMetric)
	}
	met.storeGuard.RUnlock()

	met.rateGuard.Lock()
	defer met.rateGuard.Unlock()

	if _, exists := met.rates[name]; !exists {
		return fmt.Errorf("Rate %s is already registered", name)
	}

	met.rates[name] = &rate{
		metric:     baseMetric,
		samples:    make(tcontainer.Int64Slice, numSamples),
		numMedians: int(numMedianSamples),
		lastSample: 0,
		value:      0,
		index:      0,
	}

	return nil
}

// Set sets a given metric to a given value. This operation is atomic.
func (met *Metrics) Set(name string, value int64) {
	met.storeGuard.RLock()
	defer met.storeGuard.RUnlock()
	atomic.StoreInt64(met.store[name], value)
}

// SetI is Set for int values (conversion to int64)
func (met *Metrics) SetI(name string, value int) {
	met.storeGuard.RLock()
	defer met.storeGuard.RUnlock()
	atomic.StoreInt64(met.store[name], int64(value))
}

// SetF is Set for float64 values (conversion to int64)
func (met *Metrics) SetF(name string, value float64) {
	rounded := math.Floor(value + 0.5)
	met.storeGuard.RLock()
	defer met.storeGuard.RUnlock()
	atomic.StoreInt64(met.store[name], int64(rounded))
}

// SetB is Set for boolean values (conversion to 0/1)
func (met *Metrics) SetB(name string, value bool) {
	ivalue := 0
	if value {
		ivalue = 1
	}
	met.storeGuard.RLock()
	defer met.storeGuard.RUnlock()
	atomic.StoreInt64(met.store[name], int64(ivalue))
}

// Inc adds 1 to a given metric. This operation is atomic.
func (met *Metrics) Inc(name string) {
	met.storeGuard.RLock()
	defer met.storeGuard.RUnlock()
	atomic.AddInt64(met.store[name], 1)
}

// Dec subtracts 1 from a given metric. This operation is atomic.
func (met *Metrics) Dec(name string) {
	met.storeGuard.RLock()
	defer met.storeGuard.RUnlock()
	atomic.AddInt64(met.store[name], -1)
}

// Add adds a number to a given metric. This operation is atomic.
func (met *Metrics) Add(name string, value int64) {
	met.storeGuard.RLock()
	defer met.storeGuard.RUnlock()
	atomic.AddInt64(met.store[name], value)
}

// AddI is Add for int values (conversion to int64)
func (met *Metrics) AddI(name string, value int) {
	met.storeGuard.RLock()
	defer met.storeGuard.RUnlock()
	atomic.AddInt64(met.store[name], int64(value))
}

// AddF is Add for float64 values (conversion to int64)
func (met *Metrics) AddF(name string, value float64) {
	rounded := math.Floor(value + 0.5)
	met.storeGuard.RLock()
	defer met.storeGuard.RUnlock()
	atomic.AddInt64(met.store[name], int64(rounded))
}

// Sub subtracts a number to a given metric. This operation is atomic.
func (met *Metrics) Sub(name string, value int64) {
	met.storeGuard.RLock()
	defer met.storeGuard.RUnlock()
	atomic.AddInt64(met.store[name], -value)
}

// SubI is SubI for int values (conversion to int64)
func (met *Metrics) SubI(name string, value int) {
	met.storeGuard.RLock()
	defer met.storeGuard.RUnlock()
	atomic.AddInt64(met.store[name], int64(-value))
}

// SubF is Sub for float64 values (conversion to int64)
func (met *Metrics) SubF(name string, value float64) {
	rounded := math.Floor(value + 0.5)
	met.storeGuard.RLock()
	defer met.storeGuard.RUnlock()
	atomic.AddInt64(met.store[name], int64(-rounded))
}

// Get returns the value of a given metric or rate. This operation is atomic.
// If the value does not exists error is non-nil and the returned value is 0.
func (met *Metrics) Get(name string) (int64, error) {
	met.storeGuard.RLock()
	if val, exists := met.store[name]; exists {
		met.storeGuard.RUnlock()
		return atomic.LoadInt64(val), nil
	}
	met.storeGuard.RUnlock()

	met.rateGuard.RLock()
	if rate, exists := met.rates[name]; exists {
		met.rateGuard.RUnlock()
		return int64(rate.value), nil
	}
	met.rateGuard.RUnlock()

	return 0, fmt.Errorf("Metric %s not found.", name)
}

// Dump creates a JSON string from all stored metrics.
func (met *Metrics) Dump() ([]byte, error) {
	snapshot := make(map[string]int64)

	met.storeGuard.RLock()
	for key, value := range met.store {
		snapshot[key] = atomic.LoadInt64(value)
	}
	met.storeGuard.RUnlock()

	met.rateGuard.RLock()
	for key, rate := range met.rates {
		snapshot[key] = rate.value
	}
	met.rateGuard.RUnlock()

	return json.Marshal(snapshot)
}

// ResetMetrics resets all registered key values to 0 expect for system Metrics.
// This locks all writes in the process.
func (met *Metrics) ResetMetrics() {
	met.storeGuard.Lock()
	for key := range met.store {
		switch key {
		case MetricProcessStart, MetricGoRoutines, MetricGoVersion:
			// ignore
		default:
			*met.store[key] = 0
		}
	}
	met.storeGuard.Unlock()

	met.rateGuard.Lock()
	for _, rate := range met.rates {
		rate.lastSample = 0
		rate.value = 0
		rate.samples.Set(0)
	}
	met.rateGuard.Unlock()
}

// FetchAndReset resets all of the given keys to 0 and returns the
// value before the reset as array. If a given metric does not exist
// it is ignored. This locks all writes in the process.
func (met *Metrics) FetchAndReset(keys ...string) map[string]int64 {
	state := make(map[string]int64)

	met.storeGuard.Lock()
	for _, key := range keys {
		if val, exists := met.store[key]; exists {
			state[key] = *val
			*val = 0
		}
	}
	met.storeGuard.Unlock()

	met.rateGuard.Lock()
	for _, key := range keys {
		if rate, exists := met.rates[key]; exists {
			rate.lastSample = 0
			rate.value = 0
			rate.samples.Set(0)
		}
	}
	met.rateGuard.Unlock()

	return state
}

func (met *Metrics) updateSystemMetrics() {
	met.storeGuard.RLock()
	defer met.storeGuard.RUnlock()

	stats := new(runtime.MemStats)
	runtime.ReadMemStats(stats)

	met.SetI(MetricGoRoutines, runtime.NumGoroutine())
	met.Set(MetricMemoryAllocated, int64(stats.Alloc))
	met.SetB(MetricMemoryGCEnabled, stats.EnableGC)
	met.Set(MetricMemoryNumObjects, int64(stats.HeapObjects))
}

func (met *Metrics) rateSampler() {
	for {
		_, isRunning := <-met.ticker.C
		if !isRunning {
			return // ### return, ticker stopped ###
		}

		met.updateSystemMetrics()
		met.updateRates()
	}
}

func (met *Metrics) updateRates() {
	// Read current values in a snapshot to avoid deadlocks
	snapshot := make(map[string]int64)
	met.storeGuard.RLock()
	for key, value := range met.store {
		snapshot[key] = atomic.LoadInt64(value)
	}
	met.storeGuard.RUnlock()

	met.rateGuard.Lock()
	defer met.rateGuard.Unlock()

	for _, rate := range met.rates {
		sample := snapshot[rate.metric]
		sampleIdx := rate.index % len(rate.samples)
		rate.samples[sampleIdx] = sample - rate.lastSample
		rate.index++
		rate.lastSample = sample

		// Calculate value
		switch {
		case rate.numMedians == 1:
			// Mean of all values
			total := int64(0)
			for _, v := range rate.samples {
				total += v
			}
			rate.value = total / int64(len(rate.samples))

		case rate.numMedians == 0, rate.numMedians >= len(rate.samples):
			// Median of all values
			values := make(tcontainer.Int64Slice, len(rate.samples))
			copy(values, rate.samples)
			values.Sort()
			rate.value = values[len(values)/2]

		default:
			// Median of means
			blockSize := len(rate.samples) / rate.numMedians
			blocks := make(tcontainer.Int64Slice, rate.numMedians)
			for i, v := range rate.samples {
				blocks[i/blockSize] += int64(v)
			}
			values := make(tcontainer.Int64Slice, rate.numMedians)
			for i, b := range blocks {
				offset := i * blockSize
				if offset+blockSize > len(blocks) {
					size := len(blocks) - offset
					values[i] = b / int64(size)
				} else {
					values[i] = b / int64(blockSize)
				}
			}
			values.Sort()
			rate.value = values[len(values)/2]
		}
	}
}
