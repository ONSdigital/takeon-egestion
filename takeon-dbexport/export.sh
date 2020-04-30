#!/usr/bin/env bash

echo $PWD
SRCPATH=$PWD/go/src/github.com/takeon-db-export-lambda
GOPATH=$PWD/go
# cd $SRCPATH
go get -t -v ./...
env GOOS=linux go build  -o bin/main
serverless package --package pkg
serverless deploy --function main