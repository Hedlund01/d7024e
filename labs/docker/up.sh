
if [ $# -ne 1 ]; then
    echo "Usage: $0 <number_of_instances>"
    exit 1
fi

# Get the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
COMPOSE_FILE="$SCRIPT_DIR/../docker-compose.yml"

NUMBER_OF_REPLICAS=$1

echo "Starting $NUMBER_OF_REPLICAS Kademlia node replicas..."

# Export the environment variable to override the .env file
export NUMBER_OF_REPLICAS=$NUMBER_OF_REPLICAS

docker stack deploy --compose-file "$COMPOSE_FILE" KademliaStack

docker service logs -f KademliaStack_kademliaNodes --raw