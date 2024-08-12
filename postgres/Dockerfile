# syntax=docker/dockerfile:1
FROM postgres:16.3-bookworm

ENV POSTGRES_DB=mydatabase
ENV POSTGRES_USER=myuser
ENV POSTGRES_PASSWORD=mypassword

# Copie os arquivos de inicialização (se necessário)
COPY *.sql /docker-entrypoint-initdb.d/
COPY data_files/* /docker-entrypoint-initdb.d/

EXPOSE 5432

