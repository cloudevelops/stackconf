#!/bin/bash
rm -rf /tmp/stackconf
mkdir -p /tmp/stackconf/usr/local/bin
cp -a $GOPATH/bin/stackconf /tmp/stackconf/usr/local/bin
fpm -s dir -t deb -C /tmp/stackconf --name stackconf --version 0.1.24 --description "stack orchestration engine" --package /tmp/stackconf
