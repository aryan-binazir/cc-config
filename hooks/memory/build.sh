#!/bin/bash
cd "$(dirname "$0")"
go mod tidy
go build -tags sqlite_omit_load_extension -ldflags="-s -w" -o memory memory.go types.go
go build -tags sqlite_omit_load_extension -ldflags="-s -w" -o query query.go types.go