#!/bin/bash

if [ $# -ne 1 ]; then
    echo "Usage: $0 <number_of_instances>"
    exit 1
fi

NUMBER_OF_REPLICAS=$1

echo "Starting $NUMBER_OF_REPLICAS Kademlia node replicas..."

# Export the environment variable to override the .env file
export NUMBER_OF_REPLICAS=$NUMBER_OF_REPLICAS


docker stack deploy --compose-file ../docker-compose.yml KademliaStack

docker service logs -f KademliaStack_kademliaNodes --raw