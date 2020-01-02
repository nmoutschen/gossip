#!/bin/bash

trap ctrl_c INT

function ctrl_c() {
    kill -INT $NODE1_PID $NODE2_PID
    exit
}

# Start nodes
pushd ./node/
go build .

GOSSIP_NODE_PORT=8080 ./node &
NODE1_PID=$!

GOSSIP_NODE_PORT=8081 ./node &
NODE2_PID=$!

# Waiting for the peer to come online
while ! nc -vz 127.0.0.1 8080; do
    sleep 1
done

# Send peering
curl -s -X POST -d '{"ip": "127.0.0.1", "port": 8081}' http://127.0.0.1:8080/peers
sleep 1s

if (( $(curl -s http://127.0.0.1:8080/peers | jq '.peers | length') < 1 )); then
    echo "127.0.0.1:8080 does not have any peer"
    kill -INT $NODE1_PID $NODE2_PID
    exit 1
fi

if (( $(curl -s http://127.0.0.1:8081/peers | jq '.peers | length') < 1 )); then
    echo "127.0.0.1:8081 does not have any peer"
    kill -INT $NODE1_PID $NODE2_PID
    exit 1
fi

# Shut down one peer
kill -INT $NODE1_PID
if (( $(curl -s http://127.0.0.1:8081/peers | jq '.peers | length') > 0 )); then
    echo "127.0.0.1:8081 still has peers"
    kill -INT $NODE1_PID $NODE2_PID
    exit 1
fi

# Shut down the other peer
kill -INT $NODE2_PID

echo "Peering worked as expected"