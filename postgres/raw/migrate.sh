#!/bin/bash
set -e

# Create test database and user
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
	CREATE USER test_user WITH PASSWORD 'password';
	CREATE DATABASE test_rollupsdb OWNER test_user;
	GRANT CONNECT ON DATABASE test_rollupsdb TO test_user;
	GRANT CREATE ON DATABASE test_rollupsdb TO test_user;
	GRANT TEMP ON DATABASE test_rollupsdb TO test_user;
EOSQL

# Run all SQL ignore errors
for f in /sql/*.sql; do
	psql -v ON_ERROR_STOP=0 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" -f $f
done