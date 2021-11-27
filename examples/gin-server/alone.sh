#!/bin/sh
basedir=$(dirname "$0")
$basedir/gin-server -alone -f $basedir/gin-server.cf
