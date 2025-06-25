#!/bin/bash

ENV_FILE=".env"

DB_FILE="falkie.db"

set -a
source $ENV_FILE
set +a

echo ${TELEGRAM_TOKEN}

if [[ ! -f "$DB_FILE" ]]; then
    touch "$DB_FILE"
fi

go build src/main.go
chmod +x ./main
./main