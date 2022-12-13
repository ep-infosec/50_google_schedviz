//
// Copyright 2019 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS-IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
//
package sched

import (
	"fmt"
	"sort"
	"time"

	log "github.com/golang/glog"
	"github.com/Workiva/go-datastructures/augmentedtree"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	eventpb "github.com/google/schedviz/tracedata/schedviz_events_go_proto"
	"github.com/google/schedviz/tracedata/trace"
)

// Collection provides an interface for processing EventSets containing
// scheduling information and serving queries over that scheduling information.
type Collection struct {
	stringTable *stringTable
	// If timestamp normalization is requested, the duration to subtract from each
	// event's timestamp to normalize it.  This is the actual timestamp of the
	// first valid, unclipped, sched event.
	normalizationOffset trace.Timestamp
	// A mapping from CPU to vector of threadSpans, reflecting what was
	// known to be running on each CPU at each moment.
	runningSpansByCPU map[CPUID][]*threadSpan
	// A mapping from CPU to interval tree.  Each CPU's one-dimensional interval
	// tree contains intervals during which PIDs slept, and can be queried for
	// the set of sleeping PIDs at any moment.
	sleepingSpansByCPU map[CPUID]augmentedtree.Tree
	// A mapping from CPU to interval tree.  Each CPU's one-dimensional interval
	// tree contains intervals during which PIDs waited, and can be queried for
	// the set of waiting PIDs at any moment.
	waitingSpansByCPU map[CPUID]augmentedtree.Tree
	spansByPID        map[PID][]*threadSpan
	options           *collectionOptions
	// cpus is a cached copy of all CPUs in the collection.
	cpus map[CPUID]struct{}
	// pids is a cached copy of all PIDs in the collection.
	pids map[PID]struct{}
	// Cached start and end timestamps of the collection.
	startTimestamp trace.Timestamp
	endTimestamp   trace.Timestamp
	// Trace collection containing the event set
	TraceCollection *trace.Collection
	// Thread transitions generated from the events loaded into this collection
	ThreadTransitions []*threadTransition
	// A mapping from dropped event IDs to the number of transitions that
	// dropped them.
	droppedEventCountsByID map[int]int
	// A count of the number of synthetic transitions inserted in the collection.
	syntheticTransitionCount int
}

// NewCollection builds and returns a new sched.Collection based on the ktrace
// event set in es, or nil and an error if one could not be created.  If the
// normalizeTimestamps argument is true, all valid, unclipped, sched event
// timestamps will be normalized to the first valid, unclipped, sched event's.
func NewCollection(es *eventpb.EventSet, options ...Option) (*Collection, error) {
	c := &Collection{
		normalizationOffset:    Unknown,
		runningSpansByCPU:      make(map[CPUID][]*threadSpan),
		sleepingSpansByCPU:     make(map[CPUID]augmentedtree.Tree),
		waitingSpansByCPU:      make(map[CPUID]augmentedtree.Tree),
		options:                &collectionOptions{},
		cpus:                   map[CPUID]struct{}{},
		pids:                   map[PID]struct{}{},
		droppedEventCountsByID: map[int]int{},
	}
	for _, option := range options {
		if err := option(c.options); err != nil {
			return nil, err
		}
	}
	// If no EventLoaders was specified, use the event set's default.
	if c.options.loaders == nil {
		elt := es.GetDefaultLoadersType()
		log.Infof("Using default event loader type %s", elt)
		el, err := EventLoader(elt)
		if err != nil {
			return nil, err
		}
		c.options.loaders = el
	}
	if err := c.buildSpansByPID(es, c.options.loaders); err != nil {
		return nil, err
	}
	if err := c.buildSpansByCPU(); err != nil {
		return nil, err
	}
	return c, nil
}

// buildSpansByPID loads the events in the provided EventSet as
// threadTransitions,, infers any CPU or state information they are missing,
// and convolutes them into threadSpans.
func (c *Collection) buildSpansByPID(es *eventpb.EventSet, eventLoaders map[string]func(*trace.Event, *ThreadTransitionSetBuilder) error) error {
	stringBank := newStringBank()
	c.stringTable = stringBank.stringTable
	eventLoader, err := newEventLoader(eventLoaders, stringBank)
	if err != nil {
		return fmt.Errorf("failed to use eventLoaders: %s", err)
	}
	coll, err := trace.NewCollection(es)
	if err != nil {
		return err
	}
	c.TraceCollection = coll
	var ts *threadSpanSet
	for eventIndex := 0; eventIndex < coll.EventCount(); eventIndex++ {
		ev, err := coll.EventByIndex(eventIndex)
		if err != nil {
			return err
		}
		// Bypass clipped events.
		if ev.Clipped {
			continue
		}
		// Translate the event into ThreadTransitions.
		tts, err := eventLoader.threadTransitions(ev)
		if err != nil {
			return err
		}
		// Skip events that do not make transitions.
		if len(tts) == 0 {
			continue
		}
		// If the normalization offset hasn't yet been set, do so.
		if c.normalizationOffset == UnknownTimestamp {
			if c.options.normalizeTimestamps {
				c.normalizationOffset = ev.Timestamp
			} else {
				c.normalizationOffset = 0
			}
			c.startTimestamp = ev.Timestamp - c.NormalizationOffset()
		}
		// Initialize the threadIntervalBuilder on first use.
		if ts == nil {
			ts = newThreadSpanSet(c.startTimestamp, c.options)
		}
		c.endTimestamp = ev.Timestamp - c.NormalizationOffset()
		c.ThreadTransitions = tts
		// Add those transitions to the threadIntervalBuilder.
		for _, tt := range tts {
			tt.Timestamp -= c.normalizationOffset
			if err := ts.addTransition(tt); err != nil {
				return err
			}
		}
	}
	if ts == nil {
		return status.Errorf(codes.InvalidArgument, "no usable events in collection")
	}
	c.spansByPID, err = ts.threadSpans(c.endTimestamp)
	c.droppedEventCountsByID = ts.droppedEventCountsByID
	c.syntheticTransitionCount = ts.syntheticTransitionCount
	return err
}

// buildSpansByCPU iterates over all per-PID threadSpans, assembling a new
// slice of running spans and interval trees of all sleeping and waiting spans
// for each CPU.  It must be invoked after buildSpansByPID has successfully
// completed.
func (c *Collection) buildSpansByCPU() error {
	cib := newCPUSpanSet()
	for _, spans := range c.spansByPID {
		for _, span := range spans {
			if span.cpu == UnknownCPU || span.pid == UnknownPID {
				continue
			}
			c.pids[span.pid] = struct{}{}
			c.cpus[span.cpu] = struct{}{}
			cib.addSpan(span)
		}
	}
	var err error
	c.runningSpansByCPU, c.sleepingSpansByCPU, c.waitingSpansByCPU, err = cib.cpuTrees()
	return err
}

// LookupCommand returns the command for the provided stringID.  If the provided
// stringID does not have a valid lookup,
func (c *Collection) LookupCommand(command stringID) (string, error) {
	if command == UnknownCommand {
		return "<unknown>", nil
	}
	commStr, err := c.stringTable.stringByID(command)
	if err != nil {
		return "", status.Errorf(codes.Internal, "failed to find command for id %d", command)
	}
	return commStr, nil
}

func cpuLookupFunc(ev *trace.Event) ([]CPUID, error) {
	if ev.Clipped {
		return nil, nil
	}
	switch ev.Name {
	case "sched_migrate_task":
		prevCPU, ok := ev.NumberProperties["orig_cpu"]
		if !ok {
			return nil, status.Errorf(codes.Internal, "sched_migrate_task lacks orig_cpu field")
		}
		return []CPUID{CPUID(ev.CPU), CPUID(prevCPU)}, nil
	case "sched_wakeup", "sched_wakeup_new":
		targetCPU, ok := ev.NumberProperties["target_cpu"]
		if !ok {
			return nil, status.Errorf(codes.Internal, "%s lacks target_cpu field", ev.Name)
		}
		return []CPUID{CPUID(targetCPU)}, nil
	}
	return []CPUID{CPUID(ev.CPU)}, nil
}

// GetRawEvents returns the raw events in this collection
// Timestamps are normalized if timestamp normalization is enabled on the collection.
func (c *Collection) GetRawEvents(filters ...Filter) ([]*trace.Event, error) {
	perCPUColl, err := NewPerCPUCollection(c, cpuLookupFunc)
	if err != nil {
		return nil, err
	}

	var events = []*trace.Event{}

	for _, eventIndex := range perCPUColl.EventIndices(filters...) {
		ev, err := c.TraceCollection.EventByIndex(eventIndex)
		if err != nil {
			return nil, err
		}
		// Adjust the event's timestamp.
		ev.Timestamp = ev.Timestamp - c.normalizationOffset
		events = append(events, ev)
	}

	return events, nil
}

// Interval returns the first and last timestamps of the events present in this
// Collection.  Only valid if tc.Valid() is true.
// FILTERS:
//   TimeRange, StartTimestamp, EndTimestamp: The returned range is clipped to
//       the filtered-in range.
func (c *Collection) Interval(filters ...Filter) (startTS, endTS trace.Timestamp) {
	f := buildFilter(c, filters)
	return f.startTimestamp, f.endTimestamp
}

// CPUs returns the CPUs that the collection covers.
// FILTERS:
//   CPUs: the returned set of CPUs is clipped to the filtered-in set of CPUs.
func (c *Collection) CPUs(filters ...Filter) map[CPUID]struct{} {
	f := buildFilter(c, filters)
	return f.cpus
}

// NormalizationOffset returns the duration to subtract from each event's
// timestamp to normalize it.  This is the actual timestamp of the first
// valid, unclipped, sched event if timestamp normalization is enabled, or
// zero if it is not.
func (c *Collection) NormalizationOffset() trace.Timestamp {
	return c.normalizationOffset
}

// TimestampFromDuration returns a trace.Timestamp corresponding to a
// time.Duration from the trace's start.
func (c *Collection) TimestampFromDuration(dur time.Duration) trace.Timestamp {
	return trace.Timestamp(dur.Nanoseconds())
}

// DurationFromTimestamp returns a time.Duration from the trace's start
// corresponding to a trace.Timestamp.
func (c *Collection) DurationFromTimestamp(ts trace.Timestamp) time.Duration {
	return time.Duration(ts) * time.Nanosecond
}

// DurationFromSchedDuration returns a time.Duration corresponding to a
// sched.Duration.
func (c *Collection) DurationFromSchedDuration(dur Duration) time.Duration {
	return time.Duration(dur) * time.Nanosecond
}

// DroppedEventIDs returns the IDs, or indices, of events dropped during CPU
// and state inference.
func (c *Collection) DroppedEventIDs() []int {
	var ret = []int{}
	for eventID := range c.droppedEventCountsByID {
		ret = append(ret, eventID)
	}
	sort.Slice(ret, func(a, b int) bool {
		return ret[a] < ret[b]
	})
	return ret
}

// SyntheticTransitionCount returns the number of synthetic thread transitions
// (or synthetic scheduling events) needed to be inserted to interpret the
// collection.
func (c *Collection) SyntheticTransitionCount() int {
	return c.syntheticTransitionCount
}

// ExpandCPUs takes a slice of CPUs, and either returns that slice,
// if it is not empty, or if it is empty, a slice of all CPUs in the
// collection.
func (c *Collection) ExpandCPUs(cpus []int64) []int64 {
	if len(cpus) == 0 {
		// Return a slice of CPUs observed in the collection, in increasing order.
		var cpus = []int64{}
		for cpu := range c.CPUs() {
			cpus = append(cpus, int64(cpu))
		}
		sort.Slice(cpus, func(i, j int) bool {
			return cpus[i] < cpus[j]
		})

		return cpus
	}
	return cpus
}
