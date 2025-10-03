#!/usr/bin/env bash
set -e

echo "mode: atomic" > coverage.out
for pkg in $(go list ./... | grep -v /vendor/); do
    go test -covermode=atomic -coverprofile=profile.out $pkg
    if [ -f profile.out ]; then
        tail -n +2 profile.out >> coverage.out
        rm profile.out
    fi
done