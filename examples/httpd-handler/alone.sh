#!/bin/sh
basedir=$(dirname "$0")
$basedir/httpd-handler -alone -f $basedir/httpd-handler.cf
