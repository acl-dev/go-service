#!/bin/sh

protoc --go_out=plugins=grpc:../pb ./hello.proto
