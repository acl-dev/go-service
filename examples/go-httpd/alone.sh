#!/bin/sh
basedir=$(dirname "$0")
$basedir/go-httpd -alone -f $basedir/go-httpd.cf
