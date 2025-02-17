# dump all the containers
docker system prune -a docker volume prune

# Start postgres
docker-compose up -d postgres

# start redis
docker-compose up -d redis

# start app
docker-compose up -d orus-pay

# Add admin
docker-compose run --rm admin-seed