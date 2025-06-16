// Command execution_log_compact_to_csv converts Bazel's compact execution log to CSV.
// Usage example:
//
//	go run execution_log_compact_to_csv.go --input_execlog=compact_execlog --include ^//
package main

import (
	"bufio"
	"encoding/csv"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"google.golang.org/protobuf/proto"
)

func decodeVarint32(r io.Reader) (uint32, error) {
	var buf [1]byte
	var result uint32
	var shift uint
	for {
		if shift >= 32 {
			return 0, errors.New("varint32 overflow")
		}
		_, err := r.Read(buf[:])
		if err != nil {
			return 0, err
		}
		b := buf[0]
		result |= uint32(b&0x7F) << shift
		if b < 0x80 {
			break
		}
		shift += 7
	}
	return result, nil
}

func main() {
	inputExeclog := flag.String("input_execlog", "", "Path to the zstd _decompressed_ input file as generated with bazel's --execution_log_compact_file option.")
	include := flag.String("include", "", "Space separated list of regexps to match target labels to include in the CSV. If omitted, all targets are included.")
	exclude := flag.String("exclude", "", "Space separated list of regexps to match target labels to exclude from the CSV.")
	verbose := flag.Bool("verbose", false, "Log every ExecLogEntry to stderr.")
	flag.Parse()

	if *inputExeclog == "" {
		fmt.Fprintln(os.Stderr, "--input_execlog is required")
		os.Exit(1)
	}

	includeRegexps := []*regexp.Regexp{}
	if *include != "" {
		for _, s := range strings.Fields(*include) {
			re, err := regexp.Compile(s)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Invalid include regexp: %v\n", err)
				os.Exit(1)
			}
			includeRegexps = append(includeRegexps, re)
		}
	}
	excludeRegexps := []*regexp.Regexp{}
	if *exclude != "" {
		for _, s := range strings.Fields(*exclude) {
			re, err := regexp.Compile(s)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Invalid exclude regexp: %v\n", err)
				os.Exit(1)
			}
			excludeRegexps = append(excludeRegexps, re)
		}
	}

	f, err := os.Open(*inputExeclog)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open input: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()
	reader := bufio.NewReader(f)
	writer := csv.NewWriter(os.Stdout)
	defer writer.Flush()

	idToEntry := make(map[uint32]*ExecLogEntry)

	for {
		msgLen, err := decodeVarint32(reader)
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to decode varint32: %v\n", err)
			os.Exit(1)
		}
		msgBuf := make([]byte, msgLen)
		if _, err := io.ReadFull(reader, msgBuf); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to read message: %v\n", err)
			os.Exit(1)
		}

		var entry ExecLogEntry
		if err := proto.Unmarshal(msgBuf, &entry); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to unmarshal ExecLogEntry: %v\n", err)
			os.Exit(1)
		}

		if *verbose {
			fmt.Fprintf(os.Stderr, "%v\n", &entry)
		}

		switch t := entry.GetType().(type) {
		case *ExecLogEntry_File_:
			idToEntry[entry.GetId()] = &entry

		case *ExecLogEntry_Directory_:
			idToEntry[entry.GetId()] = &entry

		case *ExecLogEntry_Spawn_:
			label := t.Spawn.GetTargetLabel()

			includeMatch := len(includeRegexps) == 0
			for _, re := range includeRegexps {
				if re.MatchString(label) {
					includeMatch = true
					break
				}
			}
			excludeMatch := false
			for _, re := range excludeRegexps {
				if re.MatchString(label) {
					excludeMatch = true
					break
				}
			}
			if !includeMatch || excludeMatch {
				continue
			}

			for _, output := range t.Spawn.GetOutputs() {
				switch ot := output.Type.(type) {
				case *ExecLogEntry_Output_OutputId:
					outputEntry, ok := idToEntry[ot.OutputId]
					if !ok {
						fmt.Fprintf(os.Stderr, "Missing output entry for id %d\n", ot.OutputId)
						continue
					}
					switch oet := outputEntry.GetType().(type) {
					case *ExecLogEntry_File_:
						_ = writer.Write([]string{label, oet.File.GetPath(), oet.File.GetDigest().GetHash()})
					case *ExecLogEntry_Directory_:
						dirPath := oet.Directory.GetPath()
						for _, dirFile := range oet.Directory.GetFiles() {
							_ = writer.Write([]string{label, dirPath + "/" + dirFile.GetPath(), dirFile.GetDigest().GetHash()})
						}
					default:
						fmt.Fprintf(os.Stderr, "Unexpected output entry type (inner)\n")
					}
				}
			}
		}
	}
}
