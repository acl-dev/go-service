#!/bin/sh
basedir=$(dirname "$0")
$basedir/go-echod -alone -f $basedir/go-echod.cf
