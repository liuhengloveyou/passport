#!/bin/bash

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w"

scp ./migration ubuntu@111.231.11.139:/tmp/migration
