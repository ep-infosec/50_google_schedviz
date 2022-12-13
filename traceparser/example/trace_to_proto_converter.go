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
// Binary trace_to_proto_converter is a command line tool to convert a recorded trace into a proto.
package main

import (
	"bufio"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"flag"
	
	log "github.com/golang/glog"
	"github.com/golang/protobuf/proto"
	"github.com/google/schedviz/traceparser/traceparser"
)

var (
	formatFilePaths          = flag.String("format_files", "", "Required. Comma separated list of paths to format files. Must include path to header_page file as well.")
	traceFilesPath           = flag.String("trace_files", "", "Required. Path to the recorded trace files. Should be a folder containing cpu0, cpu1, ... files")
	outputPath               = flag.String("output_path", "", "Required. Path to the file where the output should be written.")
	statsFilesPath           = flag.String("stats_files", "", "Optional. Path to the recorded cpu stats. Should be a folder containing cpu0, cpu1, ... files")
	outputFormat             = flag.String("output_format", "proto", "Optional. Format to write the output in. Can be either \"proto\" or \"textproto\". Will use \"proto\" if not specified")
	failOnUnknownEventFormat = flag.Bool("fail_on_unknown_event_format", true, "Whether or not to continue parsing when an unknown event is encountered")
)

func main() {
		flag.Parse()

	// Filter out empty strings
	formatFilePathsSlice := strings.Split(*formatFilePaths, ",")
	for i, ffp := range formatFilePathsSlice {
		formatFilePathsSlice[i] = strings.TrimSpace(ffp)
	}
	filteredFormatFilePaths := filterSlice(formatFilePathsSlice, func(s string) bool {
		return s != ""
	})

	if len(filteredFormatFilePaths) < 2 {
		log.Exit("format_files is required. Must pass the path to at least one format file and the header_page format file.")
	}
	if *outputPath == "" {
		log.Exit("output_path is required.")
	}

	var formatFiles = make([]string, len(filteredFormatFilePaths)-1)
	var headerContent string

	i := 0
	for _, formatFilePath := range filteredFormatFilePaths {
		buf, err := ioutil.ReadFile(formatFilePath)
		if err != nil {
			log.Exitf("Failed to read format file: %s", err)
		}
		if strings.HasSuffix(formatFilePath, "header_page") {
			headerContent = string(buf)
			continue
		} else {
			formatFiles[i] = string(buf)
		}
		i++
	}

	if headerContent == "" {
		log.Exit("Must pass a path to the header_page format file using the format_files argument.")
	}

	traceParser, err := traceparser.New(headerContent, formatFiles)
	if err != nil {
		log.Exitf("Failed to parse formats: %s", err)
	}
	traceParser.SetFailOnUnknownEventFormat(*failOnUnknownEventFormat)

	traceFiles, err := ioutil.ReadDir(*traceFilesPath)
	if err != nil {
		log.Exitf("Failed to get list of trace files: %s", err)
	}

	statsFiles := []os.FileInfo{}
	if *statsFilesPath != "" {
		statsFiles, err = ioutil.ReadDir(*statsFilesPath)
		if err != nil {
			log.Exitf("Failed to get list of stats files: %s", err)
		}
	}

	eventSetBuilder := traceparser.NewEventSetBuilder(&traceParser)

	overflowedCPUs := map[int64]struct{}{}
	for i, statsFilePath := range statsFiles {
		statsFile, err := os.Open(path.Join(*traceFilesPath, statsFilePath.Name()))
		if err != nil {
			log.Exitf("Failed to open trace file: %s", err)
		}
		reader := bufio.NewReader(statsFile)
		if overflowed, err := traceparser.CPUOverflowed(reader); err != nil && overflowed {
			overflowedCPUs[int64(i)] = struct{}{}
		}
	}
	eventSetBuilder.SetOverflowedCPUs(overflowedCPUs)

	for i, traceFilePath := range traceFiles {
		traceFile, err := os.Open(path.Join(*traceFilesPath, traceFilePath.Name()))
		if err != nil {
			log.Exitf("Failed to open trace file: %s", err)
		}
		reader := bufio.NewReader(traceFile)
		if err := traceParser.ParseTrace(reader, int64(i), func(traceEvent *traceparser.TraceEvent) (bool, error) {
			if err := eventSetBuilder.AddTraceEvent(traceEvent); err != nil {
				log.Exitf("Error in AddTraceEvent: %s", err)
			}
			return true, nil
		}); err != nil {
			log.Exitf("Failed to parse trace: %s", err)
		}
	}

	protos, err := eventSetBuilder.Finalize()

	if err != nil {
		log.Exitf("Failed to finalize events: %s", err)
	}

	var output []byte
	switch *outputFormat {
	case "proto":
		output, err = proto.Marshal(protos)
		if err != nil {
			log.Exitf("Error marshalling proto. Caused by: %s", err)
		}
	case "textproto":
		output = []byte(proto.MarshalTextString(protos))
	default:
		log.Exitf("Unknown output format: %s", *outputFormat)
	}

	if err := ioutil.WriteFile(*outputPath, output, 0644); err != nil {
		log.Exitf("Error writing to output file. Caused by: %s", err)
	}
}

func filterSlice(slice []string, pred func(string) bool) []string {
	var filtered []string
	for _, s := range slice {
		if pred(s) {
			filtered = append(filtered, s)
		}
	}
	return filtered
}
