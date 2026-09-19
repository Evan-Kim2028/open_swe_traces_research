#!/usr/bin/env bash
while true; do bash "$(dirname "$0")/docker_cleanup.sh"; sleep 1800; done
