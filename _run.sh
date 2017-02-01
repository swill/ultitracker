#!/usr/bin/env bash

# build the theme components (first arg = theme name)
source __team.sh

# build and run the application
export GOOGLE_APPLICATION_CREDENTIALS=google-service-account.json
esc -o static.go static
go build
./ultitracker