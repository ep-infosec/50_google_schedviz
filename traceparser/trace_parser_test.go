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
package traceparser

import (
	"bufio"
	"encoding/gob"
	"fmt"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/schedviz/testhelpers/testhelpers"
)

var tp = &TraceParser{
	HeaderFormat: Format{
		Fields: []*FormatField{
			{FieldType: "u64 timestamp", Name: "timestamp", ProtoType: "int64", Size: 8, NumElements: 1, ElementSize: 8},
			{FieldType: "local_t commit", Name: "commit", ProtoType: "int64", Offset: 8, Size: 8, NumElements: 1, ElementSize: 8, Signed: true},
			{FieldType: "int overwrite", Name: "overwrite", ProtoType: "int64", Offset: 8, Size: 1, NumElements: 1, ElementSize: 1, Signed: true},
			{FieldType: "char data", Name: "data", ProtoType: "string", Offset: 16, Size: 4080, NumElements: 1, ElementSize: 4080, Signed: true},
		},
	},

	Formats: map[uint16]*EventFormat{
		296: {
			Name: "sched_migrate_task",
			ID:   296,
			Format: Format{
				CommonFields: []*FormatField{
					{FieldType: "unsigned short common_type", Name: "common_type", ProtoType: "int64", Size: 2, NumElements: 1, ElementSize: 2},
					{FieldType: "unsigned char common_flags", Name: "common_flags", ProtoType: "string", Offset: 2, Size: 1, NumElements: 1, ElementSize: 1},
					{FieldType: "unsigned char common_preempt_count", Name: "common_preempt_count", ProtoType: "string", Offset: 3, Size: 1, NumElements: 1, ElementSize: 1},
					{FieldType: "int common_pid", Name: "common_pid", ProtoType: "int64", Offset: 4, Size: 4, NumElements: 1, ElementSize: 4, Signed: true},
				},
				Fields: []*FormatField{
					{FieldType: "char comm[16]", Name: "comm", ProtoType: "string", Offset: 8, Size: 16, NumElements: 16, ElementSize: 1, Signed: true},
					{FieldType: "pid_t pid", Name: "pid", ProtoType: "int64", Offset: 24, Size: 4, NumElements: 1, ElementSize: 4, Signed: true},
					{FieldType: "int prio", Name: "prio", ProtoType: "int64", Offset: 28, Size: 4, NumElements: 1, ElementSize: 4, Signed: true},
					{FieldType: "int orig_cpu", Name: "orig_cpu", ProtoType: "int64", Offset: 32, Size: 4, NumElements: 1, ElementSize: 4, Signed: true},
					{FieldType: "int dest_cpu", Name: "dest_cpu", ProtoType: "int64", Offset: 36, Size: 4, NumElements: 1, ElementSize: 4, Signed: true},
				},
			},
		},
		297: {
			Name: "sched_switch",
			ID:   297,
			Format: Format{
				CommonFields: []*FormatField{
					{FieldType: "unsigned short common_type", Name: "common_type", ProtoType: "int64", Size: 2, NumElements: 1, ElementSize: 2},
					{FieldType: "unsigned char common_flags", Name: "common_flags", ProtoType: "string", Offset: 2, Size: 1, NumElements: 1, ElementSize: 1},
					{FieldType: "unsigned char common_preempt_count", Name: "common_preempt_count", ProtoType: "string", Offset: 3, Size: 1, NumElements: 1, ElementSize: 1},
					{FieldType: "int common_pid", Name: "common_pid", ProtoType: "int64", Offset: 4, Size: 4, NumElements: 1, ElementSize: 4, Signed: true},
				},
				Fields: []*FormatField{
					{FieldType: "char prev_comm[16]", Name: "prev_comm", ProtoType: "string", Offset: 8, Size: 16, NumElements: 16, ElementSize: 1, Signed: true},
					{FieldType: "pid_t prev_pid", Name: "prev_pid", ProtoType: "int64", Offset: 24, Size: 4, NumElements: 1, ElementSize: 4, Signed: true},
					{FieldType: "int prev_prio", Name: "prev_prio", ProtoType: "int64", Offset: 28, Size: 4, NumElements: 1, ElementSize: 4, Signed: true},
					{FieldType: "long prev_prio", Name: "prev_state", ProtoType: "int64", Offset: 32, Size: 8, NumElements: 1, ElementSize: 8, Signed: true},
					{FieldType: "char next_comm[16]", Name: "next_comm", ProtoType: "string", Offset: 40, Size: 16, NumElements: 16, ElementSize: 1, Signed: true},
					{FieldType: "pid_t next_pid", Name: "next_pid", ProtoType: "int64", Offset: 56, Size: 4, NumElements: 1, ElementSize: 4, Signed: true},
					{FieldType: "int next_prio", Name: "next_prio", ProtoType: "int64", Offset: 60, Size: 4, NumElements: 1, ElementSize: 4, Signed: true},
				},
			},
		},
		298: {
			Name: "sched_wakeup_new",
			ID:   298,
			Format: Format{
				CommonFields: []*FormatField{
					{FieldType: "unsigned short common_type", Name: "common_type", ProtoType: "int64", Size: 2, NumElements: 1, ElementSize: 2},
					{FieldType: "unsigned char common_flags", Name: "common_flags", ProtoType: "string", Offset: 2, Size: 1, NumElements: 1, ElementSize: 1},
					{FieldType: "unsigned char common_preempt_count", Name: "common_preempt_count", ProtoType: "string", Offset: 3, Size: 1, NumElements: 1, ElementSize: 1},
					{FieldType: "int common_pid", Name: "common_pid", ProtoType: "int64", Offset: 4, Size: 4, NumElements: 1, ElementSize: 4, Signed: true},
				},
				Fields: []*FormatField{
					{FieldType: "char comm[16]", Name: "comm", ProtoType: "string", Offset: 8, Size: 16, NumElements: 16, ElementSize: 1, Signed: true},
					{FieldType: "pid_t pid", Name: "pid", ProtoType: "int64", Offset: 24, Size: 4, NumElements: 1, ElementSize: 4, Signed: true},
					{FieldType: "int prio", Name: "prio", ProtoType: "int64", Offset: 28, Size: 4, NumElements: 1, ElementSize: 4, Signed: true},
					{FieldType: "int success", Name: "success", ProtoType: "int64", Offset: 32, Size: 4, NumElements: 1, ElementSize: 4, Signed: true},
					{FieldType: "int target_cpu", Name: "target_cpu", ProtoType: "int64", Offset: 36, Size: 4, NumElements: 1, ElementSize: 4, Signed: true},
				},
			},
		},
		299: {
			Name: "sched_wakeup",
			ID:   299,
			Format: Format{
				CommonFields: []*FormatField{
					{FieldType: "unsigned short common_type", Name: "common_type", ProtoType: "int64", Size: 2, NumElements: 1, ElementSize: 2},
					{FieldType: "unsigned char common_flags", Name: "common_flags", ProtoType: "string", Offset: 2, Size: 1, NumElements: 1, ElementSize: 1},
					{FieldType: "unsigned char common_preempt_count", Name: "common_preempt_count", ProtoType: "string", Offset: 3, Size: 1, NumElements: 1, ElementSize: 1},
					{FieldType: "int common_pid", Name: "common_pid", ProtoType: "int64", Offset: 4, Size: 4, NumElements: 1, ElementSize: 4, Signed: true},
				},
				Fields: []*FormatField{
					{FieldType: "char comm[16]", Name: "comm", ProtoType: "string", Offset: 8, Size: 16, NumElements: 16, ElementSize: 1, Signed: true},
					{FieldType: "pid_t pid", Name: "pid", ProtoType: "int64", Offset: 24, Size: 4, NumElements: 1, ElementSize: 4, Signed: true},
					{FieldType: "int prio", Name: "prio", ProtoType: "int64", Offset: 28, Size: 4, NumElements: 1, ElementSize: 4, Signed: true},
					{FieldType: "int success", Name: "success", ProtoType: "int64", Offset: 32, Size: 4, NumElements: 1, ElementSize: 4, Signed: true},
					{FieldType: "int target_cpu", Name: "target_cpu", ProtoType: "int64", Offset: 36, Size: 4, NumElements: 1, ElementSize: 4, Signed: true},
				},
			},
		},
	},
	failOnUnknownEventFormat: true,
}

var tp32 = &TraceParser{
	HeaderFormat: Format{
		Fields: []*FormatField{
			{FieldType: "u64 timestamp", Name: "timestamp", ProtoType: "int64", Size: 8, NumElements: 1, ElementSize: 8},
			{FieldType: "local_t commit", Name: "commit", ProtoType: "int64", Offset: 8, Size: 4, NumElements: 1, ElementSize: 4, Signed: true},
			{FieldType: "int overwrite", Name: "overwrite", ProtoType: "int64", Offset: 8, Size: 1, NumElements: 1, ElementSize: 1, Signed: true},
			{FieldType: "char data", Name: "data", ProtoType: "string", Offset: 16, Size: 4084, NumElements: 1, ElementSize: 4084, Signed: true},
		},
	},
	Formats: map[uint16]*EventFormat{
		220: {
			Name: "sched_migrate_task",
			ID:   220,
			Format: Format{
				CommonFields: []*FormatField{
					{FieldType: "unsigned short common_type", Name: "common_type", ProtoType: "int64", Size: 2, NumElements: 1, ElementSize: 2},
					{FieldType: "unsigned char common_flags", Name: "common_flags", ProtoType: "string", Offset: 2, Size: 1, NumElements: 1, ElementSize: 1},
					{FieldType: "unsigned char common_preempt_count", Name: "common_preempt_count", ProtoType: "string", Offset: 3, Size: 1, NumElements: 1, ElementSize: 1},
					{FieldType: "int common_pid", Name: "common_pid", ProtoType: "int64", Offset: 4, Size: 4, NumElements: 1, ElementSize: 4, Signed: true},
					{FieldType: "unsigned short common_migrate_disable", Name: "common_migrate_disable", ProtoType: "int64", Offset: 8, Size: 2, NumElements: 1, ElementSize: 2, Signed: true},
					{FieldType: "unsigned short common_padding", Name: "common_padding", ProtoType: "int64", Offset: 10, Size: 2, NumElements: 1, ElementSize: 2, Signed: true},
				},
				Fields: []*FormatField{
					{FieldType: "char comm[16]", Name: "comm", ProtoType: "string", Offset: 16, Size: 16, NumElements: 16, ElementSize: 1, Signed: true},
					{FieldType: "pid_t pid", Name: "pid", ProtoType: "int64", Offset: 32, Size: 4, NumElements: 1, ElementSize: 4, Signed: true},
					{FieldType: "int prio", Name: "prio", ProtoType: "int64", Offset: 36, Size: 4, NumElements: 1, ElementSize: 4, Signed: true},
					{FieldType: "int orig_cpu", Name: "orig_cpu", ProtoType: "int64", Offset: 40, Size: 4, NumElements: 1, ElementSize: 4, Signed: true},
					{FieldType: "int dest_cpu", Name: "dest_cpu", ProtoType: "int64", Offset: 44, Size: 4, NumElements: 1, ElementSize: 4, Signed: true},
				},
			},
		},
		221: {
			Name: "sched_switch",
			ID:   221,
			Format: Format{
				CommonFields: []*FormatField{
					{FieldType: "unsigned short common_type", Name: "common_type", ProtoType: "int64", Size: 2, NumElements: 1, ElementSize: 2},
					{FieldType: "unsigned char common_flags", Name: "common_flags", ProtoType: "string", Offset: 2, Size: 1, NumElements: 1, ElementSize: 1},
					{FieldType: "unsigned char common_preempt_count", Name: "common_preempt_count", ProtoType: "string", Offset: 3, Size: 1, NumElements: 1, ElementSize: 1},
					{FieldType: "int common_pid", Name: "common_pid", ProtoType: "int64", Offset: 4, Size: 4, NumElements: 1, ElementSize: 4, Signed: true},
					{FieldType: "unsigned short common_migrate_disable", Name: "common_migrate_disable", ProtoType: "int64", Offset: 8, Size: 2, NumElements: 1, ElementSize: 2, Signed: true},
					{FieldType: "unsigned short common_padding", Name: "common_padding", ProtoType: "int64", Offset: 10, Size: 2, NumElements: 1, ElementSize: 2, Signed: true},
				},
				Fields: []*FormatField{
					{FieldType: "char prev_comm[16]", Name: "prev_comm", ProtoType: "string", Offset: 16, Size: 16, NumElements: 16, ElementSize: 1, Signed: true},
					{FieldType: "pid_t prev_pid", Name: "prev_pid", ProtoType: "int64", Offset: 32, Size: 4, NumElements: 1, ElementSize: 4, Signed: true},
					{FieldType: "int prev_prio", Name: "prev_prio", ProtoType: "int64", Offset: 36, Size: 4, NumElements: 1, ElementSize: 4, Signed: true},
					{FieldType: "long prev_prio", Name: "prev_state", ProtoType: "int64", Offset: 40, Size: 4, NumElements: 1, ElementSize: 8, Signed: true},
					{FieldType: "char next_comm[16]", Name: "next_comm", ProtoType: "string", Offset: 44, Size: 16, NumElements: 16, ElementSize: 1, Signed: true},
					{FieldType: "pid_t next_pid", Name: "next_pid", ProtoType: "int64", Offset: 60, Size: 4, NumElements: 1, ElementSize: 4, Signed: true},
					{FieldType: "int next_prio", Name: "next_prio", ProtoType: "int64", Offset: 64, Size: 4, NumElements: 1, ElementSize: 4, Signed: true},
				},
			},
		},
		222: {
			Name: "sched_wakeup_new",
			ID:   222,
			Format: Format{
				CommonFields: []*FormatField{
					{FieldType: "unsigned short common_type", Name: "common_type", ProtoType: "int64", Size: 2, NumElements: 1, ElementSize: 2},
					{FieldType: "unsigned char common_flags", Name: "common_flags", ProtoType: "string", Offset: 2, Size: 1, NumElements: 1, ElementSize: 1},
					{FieldType: "unsigned char common_preempt_count", Name: "common_preempt_count", ProtoType: "string", Offset: 3, Size: 1, NumElements: 1, ElementSize: 1},
					{FieldType: "int common_pid", Name: "common_pid", ProtoType: "int64", Offset: 4, Size: 4, NumElements: 1, ElementSize: 4, Signed: true},
					{FieldType: "unsigned short common_migrate_disable", Name: "common_migrate_disable", ProtoType: "int64", Offset: 8, Size: 2, NumElements: 1, ElementSize: 2, Signed: true},
					{FieldType: "unsigned short common_padding", Name: "common_padding", ProtoType: "int64", Offset: 10, Size: 2, NumElements: 1, ElementSize: 2, Signed: true},
				},
				Fields: []*FormatField{
					{FieldType: "char comm[16]", Name: "comm", ProtoType: "string", Offset: 16, Size: 16, NumElements: 16, ElementSize: 1, Signed: true},
					{FieldType: "pid_t pid", Name: "pid", ProtoType: "int64", Offset: 32, Size: 4, NumElements: 1, ElementSize: 4, Signed: true},
					{FieldType: "int prio", Name: "prio", ProtoType: "int64", Offset: 36, Size: 4, NumElements: 1, ElementSize: 4, Signed: true},
					{FieldType: "int success", Name: "success", ProtoType: "int64", Offset: 40, Size: 4, NumElements: 1, ElementSize: 4, Signed: true},
					{FieldType: "int target_cpu", Name: "target_cpu", ProtoType: "int64", Offset: 44, Size: 4, NumElements: 1, ElementSize: 4, Signed: true},
				},
			},
		},
		223: {
			Name: "sched_wakeup",
			ID:   223,
			Format: Format{
				CommonFields: []*FormatField{
					{FieldType: "unsigned short common_type", Name: "common_type", ProtoType: "int64", Size: 2, NumElements: 1, ElementSize: 2},
					{FieldType: "unsigned char common_flags", Name: "common_flags", ProtoType: "string", Offset: 2, Size: 1, NumElements: 1, ElementSize: 1},
					{FieldType: "unsigned char common_preempt_count", Name: "common_preempt_count", ProtoType: "string", Offset: 3, Size: 1, NumElements: 1, ElementSize: 1},
					{FieldType: "int common_pid", Name: "common_pid", ProtoType: "int64", Offset: 4, Size: 4, NumElements: 1, ElementSize: 4, Signed: true},
					{FieldType: "unsigned short common_migrate_disable", Name: "common_migrate_disable", ProtoType: "int64", Offset: 8, Size: 2, NumElements: 1, ElementSize: 2, Signed: true},
					{FieldType: "unsigned short common_padding", Name: "common_padding", ProtoType: "int64", Offset: 10, Size: 2, NumElements: 1, ElementSize: 2, Signed: true},
				},
				Fields: []*FormatField{
					{FieldType: "char comm[16]", Name: "comm", ProtoType: "string", Offset: 8, Size: 16, NumElements: 16, ElementSize: 1, Signed: true},
					{FieldType: "pid_t pid", Name: "pid", ProtoType: "int64", Offset: 24, Size: 4, NumElements: 1, ElementSize: 4, Signed: true},
					{FieldType: "int prio", Name: "prio", ProtoType: "int64", Offset: 28, Size: 4, NumElements: 1, ElementSize: 4, Signed: true},
					{FieldType: "int success", Name: "success", ProtoType: "int64", Offset: 32, Size: 4, NumElements: 1, ElementSize: 4, Signed: true},
					{FieldType: "int target_cpu", Name: "target_cpu", ProtoType: "int64", Offset: 36, Size: 4, NumElements: 1, ElementSize: 4, Signed: true},
				},
			},
		},
	},
}

// readGob reads a gob file at filePath and stores its contents in obj
func readGob(t *testing.T, filePath string, obj interface{}) error {
	t.Helper()

	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()
	decoder := gob.NewDecoder(file)
	return decoder.Decode(obj)
}

func TestParseTrace(t *testing.T) {
	runFiles := testhelpers.GetRunFilesPath()

	tests := []struct {
		tp          *TraceParser
		gobFileName string
		cpuFileName string
	}{
		{tp: tp, gobFileName: "trace.gob", cpuFileName: "cpu0"},
		{tp: tp32, gobFileName: "trace-32.gob", cpuFileName: "cpu0-32"},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("TestParseTrace Case: %d", i), func(t *testing.T) {
			cpuFile, err := os.Open(path.Join(runFiles, "traceparser", "testdata", "input", test.cpuFileName))
			if err != nil {
				t.Fatalf("error reading test trace file. caused by: %s", err)
			}

			reader := bufio.NewReader(cpuFile)

			// Ignore error as it will never be thrown
			_ = test.tp.SetLittleEndian()

			var got = []TraceEvent{}
			if err := test.tp.ParseTrace(reader, 0 /*=cpu*/, func(event *TraceEvent) (bool, error) {
				got = append(got, *event)
				return true, nil
			}); err != nil {
				t.Fatalf("error during ParseTrace(): %s", err)
			}

			var want = []TraceEvent{}
			if err := readGob(t, path.Join(runFiles, "traceparser", "testdata", "output", test.gobFileName), &want); err != nil {
				t.Fatalf("TestParseTrace: error readed expected output file: %s", err)
			}

			if diff := cmp.Diff(want, got, cmpopts.IgnoreUnexported(TraceParser{})); diff != "" {
				t.Fatalf("TestParseTrace: Diff -want +got:\n%s", diff)
			}
		})
	}

}

type Stats struct {
	overrun, commitOverrun, droppedEvents string
}

func createStatsString(stats Stats) string {
	return fmt.Sprintf(`entries: 1945
overrun: %s
commit overrun: %s
bytes: 128768
oldest event ts: 2698497.198903
now ts: 2698499.259470
dropped events: %s
read events: 2404`, stats.overrun, stats.commitOverrun, stats.droppedEvents)
}

func TestReadStats(t *testing.T) {
	m := map[Stats]bool{
		{"1", "0", "0"}: true,
		{"0", "1", "0"}: true,
		{"0", "0", "1"}: true,
		{"1", "1", "1"}: true,
		{"0", "0", "0"}: false,
	}

	for stats, expected := range m {
		s := createStatsString(stats)
		reader := strings.NewReader(s)
		buffer := bufio.NewReader(reader)
		if clipped, err := CPUOverflowed(buffer); err != nil || clipped != expected {
			t.Fatalf("Expected clipping")
		}
	}
}
