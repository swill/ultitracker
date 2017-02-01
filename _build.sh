#!/usr/bin/env bash

# build the theme components (first arg = theme name)
source __team.sh

# build the binaries
esc -o static.go static
gox -output="bin/{{.Dir}}_{{.OS}}_{{.Arch}}" -osarch="linux/amd64 darwin/amd64"
#gox -output="bin/{{.Dir}}_{{.OS}}_{{.Arch}}" -os="linux darwin windows"