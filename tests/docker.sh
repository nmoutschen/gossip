#!/bin/bash

trap ctrl_c INT

function ctrl_c() {
    for i in {0..7}; do
        # Stop container
        docker stop gossip-node$i
    done
    docker stop gossip-control
    exit
}

# Controller node configuration
export GOSSIP_CONTROLLER_SCANINTERVAL="3s"
export GOSSIP_CONTROLLER_MINPEERS="3"

# Build images
docker build -f Dockerfile.control -t gossip-control .
docker build -f Dockerfile.node -t gossip-node .

# Start controller
docker run -d -p 7080:7080 --rm --name gossip-control \
    --env GOSSIP_CONTROLLER_SCANINTERVAL \
    --env GOSSIP_CONTROLLER_MINPEERS \
    gossip-control

# Ensure that the controller is running
while ! nc -vz 127.0.0.1 7080; do
    sleep 1
done

# Start nodes
for i in {0..7}; do
    # Start container
    docker run -d --rm --name gossip-node$i gossip-node
    # Notify controller of the new data node
    curl -s -X POST -d '{"ip": "'$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' gossip-node$i)'", "port": 8080}' http://127.0.0.1:7080/peers
done

# Wait for the controllers to connect all the nodes
sleep 10s

# Parse the graph
for l in $(curl -s http://127.0.0.1:7080/peers | jq '.nodes[].peers | length'); do
    if (( l < $GOSSIP_CONTROLLER_MINPEERS )); then
        echo "Found node with less than $GOSSIP_CONTROLLER_MINPEERS peers"
        for i in {0..7}; do
            # Stop container
            docker stop gossip-node$i
        done
        docker stop gossip-control
        exit 1
    fi
done

echo "All node have $GOSSIP_CONTROLLER_MINPEERS peers or more"
for i in {0..7}; do
    # Stop container
    docker stop gossip-node$i
done
docker stop gossip-control