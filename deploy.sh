REPO_DIR="/home/bersama/Naufal/placeholder-api"

CONTAINER_NAME="placeholder-server"

cd $REPO_DIR || exit 1

git fetch origin

LOCAL_COMMIT=$(git rev-parse @)
REMOTE_COMMIT=$(git rev-parse @{u})

if [ "$LOCAL_COMMIT" != "$REMOTE_COMMIT" ]; then
    echo "New changes found. Pulling latest changes..."

    git pull origin main

    docker-compose build placeholder-server

    docker-compose up -d placeholder-server

    echo "Deployment complete."
else
    echo "No changes detected. No action taken."
fi
