#!/bin/bash
set -e

# Create dedicated database for NocoDB to ensure schema and data isolation
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
    CREATE DATABASE "$NOCODB_DB";
EOSQL
