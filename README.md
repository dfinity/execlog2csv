# execlog2csv

The `execlog2csv` binary parses Bazel's [compact execution log](https://bazel.build/reference/command-line-reference#build-flag--execution_log_compact_file) and produces a CSV file:

```bash
$ bazel build @jemalloc//:libjemalloc --execution_log_compact_file=out.execlog.zst
$ zstd -f -d ./out.execlog.zst
$ execlog2csv --input_execlog ./out.execlog
<CSV output>

$ execlog2csv --help
Usage of execlog2csv:
  -exclude string
    	Space separated list of regexps to match target labels to exclude from the CSV.
  -include string
    	Space separated list of regexps to match target labels to include in the CSV. If omitted, all targets are included.
  -input_execlog string
    	Path to the zstd _decompressed_ input file as generated with bazel's --execution_log_compact_file option.
  -verbose
    	Log every ExecLogEntry to stderr.
```

The CSV output can be useful for debugging build determinism issues. The CSV file can be used by other tools (`awk`, `comm`, ...) for further processing.
