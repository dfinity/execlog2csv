# Development Instructions

## Build

```bash
go build .
```

## Run

```bash
bazel build @jemalloc//:libjemalloc --execution_log_compact_file=out.execlog.zst
zstd -f -d ./out.execlog.zst
go run . --input_execlog ./out.execlog
<CSV output>
```

## Regenerate the files

Dependencies:
* `protoc`
* `protoc-gen-go`


The easiest way to get all the deps and re-generate the `spawn.pb.go` file:

```bash
curl -SLO https://raw.githubusercontent.com/bazelbuild/bazel/refs/tags/7.6.0/src/main/protobuf/spawn.proto
nix shell nixpkgs#protobuf nixpkgs#protoc-gen-go --command protoc --proto_path=. --go_out=. --go_opt=paths=source_relative --go_opt=Mspawn.proto=main/ ./spawn.proto
```

## Format & Lint

```bash
go fmt . # format
golangci-lint run # run lint checks
```

## Release

Create a tag:

```shell
git tag v0.0.3
```

Push the tag to trigger release creation:

```shell
git push origin v0.0.3
```
