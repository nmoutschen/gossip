#!/bin/bash

trap ctrl_c INT

function ctrl_c() {
    kill $NODE_PIDS
    kill $CONTROL_PID
    exit
}

export NODE_PIDS=""

# Start nodes
pushd ./node/
for i in {8080..8089}; do
    GOSSIP_NODE_PORT=$i go run . &>/dev/null & export NODE_PIDS="$NODE_PIDS $!"
done
popd
echo "\$NODE_PIDS=$NODE_PIDS"

# Start controller
pushd ./control/
GOSSIP_CONTROLLER_SCANINTERVAL="10s" go run . &
export CONTROL_PID=$!
echo "\$CONTROL_PID=$CONTROL_PID"
popd

# Send peers to controller
while ! nc -vz localhost 7080; do
    sleep 1
done
for i in {8080..8089}; do
    curl -X POST -d '{"ip": "127.0.0.1", "port": '$i'}' http://127.0.0.1:7080/peers
done

while true; do sleep 5; done
