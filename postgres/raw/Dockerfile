FROM postgres:16-alpine

# Copy the SQL files to the container
RUN mkdir -v /sql
COPY *.sql /sql/

# Copy the migration script to the container
COPY migrate.sh /docker-entrypoint-initdb.d/migrate.sh
