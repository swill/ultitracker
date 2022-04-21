#!/usr/bin/env bash

# build the theme components
source __teams.sh

# build the binaries
gox -output="bin/{{.Dir}}_{{.OS}}_{{.Arch}}" -osarch="linux/amd64 darwin/amd64"