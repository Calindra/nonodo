FROM postgres:16-alpine

COPY init-test-db.sh /docker-entrypoint-initdb.d/init-test-db.sh

RUN mkdir -v /sql
COPY *.sql /sql/
