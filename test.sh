#!/bin/bash

rm -f node1.log node2.log node3.log ringrollers

go build -o ringrollers

echo "Starting 3-node ring..."
./ringrollers -id=node1 -addr=:8080 -neighbor=http://localhost:8181 -initiator=true > node1.log 2>&1 &
NODE1_PID=$!
./ringrollers -id=node2 -addr=:8181 -neighbor=http://localhost:8282 > node2.log 2>&1 &
NODE2_PID=$!
./ringrollers -id=node3 -addr=:8282 > node3.log 2>&1 &
NODE3_PID=$!

sleep 120
killall ringrollers
