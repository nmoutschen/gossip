#!/bin/bash

trap ctrl_c INT

function ctrl_c() {
    kill -INT $NODE_PIDS $CONTROL_PID
    exit
}

# Controller node configuration
export GOSSIP_CONTROLLER_SCANINTERVAL="3s"
export GOSSIP_CONTROLLER_MINPEERS="3"

# Start nodes
export NODE_PIDS=""
pushd ./node/
go build .
for i in {8080..8089}; do
    GOSSIP_NODE_PORT=$i ./node &>/dev/null & export NODE_PIDS="$NODE_PIDS $!"
done
popd
echo "\$NODE_PIDS=$NODE_PIDS"

# Start controller
pushd ./control/
go build .
./control &
export CONTROL_PID=$!
echo "\$CONTROL_PID=$CONTROL_PID"
popd

# Send peers to controller
while ! nc -vz localhost 7080; do
    sleep 1
done
for i in {8080..8089}; do
    curl -s -X POST -d '{"ip": "127.0.0.1", "port": '$i'}' http://127.0.0.1:7080/peers
done

# Wait for the controllers to connect all the nodes
sleep 7s

# Parse the graph
for l in $(curl -s http://127.0.0.1:7080/peers | jq '.nodes[].peers | length'); do
    if (( l < $GOSSIP_CONTROLLER_MINPEERS )); then
        echo "Found node with less than $GOSSIP_CONTROLLER_MINPEERS peers"
        kill -INT $NODE_PIDS $CONTROL_PID
        exit 1
    fi
done

echo "All node have $GOSSIP_CONTROLLER_MINPEERS peers or more"
kill -INT $NODE_PIDS $CONTROL_PID