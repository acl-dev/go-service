#!/bin/sh
basedir=$(dirname "$0")
$basedir/grpc-server -alone -f $basedir/grpc-server.cf
