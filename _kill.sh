#!/usr/bin/env bash

kill $(ps aux | grep '[i]ris' | awk '{print $2}')