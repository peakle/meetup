#!/bin/bash

while true; do
  (ps -e -o pcpu,args --sort=pcpu | grep -i "meetup\|parsing" | grep -v "grep" | cut -d" " -f1-5 | tail) >> ps.log
  sleep 1
done
