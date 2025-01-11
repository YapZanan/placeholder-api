REPO_DIR="/home/bersama/Naufal/placeholder-api"

CONTAINER_NAME="placeholder-server"

cd $REPO_DIR

git fetch origin

LOCAL_COMMIT=$(git rev-parse @)
REMOTE_COMMIT=$(git rev-parse @{u})

if [ "$LOCAL_COMMIT" != "$REMOTE_COMMIT" ]; then
    echo "New changes found. Pulling latest changes..."

    git pull origin main

    docker-compose build

    docker-compose up -d

    echo "Deployment complete."
else
    echo "No changes detected. No action taken."
fi
