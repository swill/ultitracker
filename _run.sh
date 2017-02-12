#!/usr/bin/env bash

# build the theme components (first arg = theme name)
source __teams.sh

# build and run the application
esc -o static.go static && go build && ./ultitracker