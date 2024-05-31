FROM postgres

ENV POSTGRES_DB=mydatabase
ENV POSTGRES_USER=myuser
ENV POSTGRES_PASSWORD=mypassword

# Copie os arquivos de inicialização (se necessário)
COPY restore.sql /docker-entrypoint-initdb.d/
COPY data_files/* /docker-entrypoint-initdb.d/

EXPOSE 5432

