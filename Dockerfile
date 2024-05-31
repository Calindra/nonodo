# Use uma imagem base que tenha o Go instalado
FROM golang:1.21

# Instale Clang
RUN apt-get update && apt-get install -y clang


# Instale Foundry
RUN curl -L https://foundry.paradigm.xyz | bash

# Configure o PATH para incluir o binário do Foundry
ENV PATH="/root/.foundry/bin:${PATH}"
ENV POSTGRES_HOST=postgres
ENV POSTGRES_PORT=5432
ENV POSTGRES_DB=mydatabase
ENV POSTGRES_USER=myuser
ENV POSTGRES_PASSWORD=mypassword

# Verifique se o Foundry e anvil estão instalados corretamente
RUN foundryup && which anvil

# Crie um diretório de trabalho fora do GOPATH
WORKDIR /app

# Copie os arquivos go.mod e go.sum para o diretório de trabalho
COPY ../go.mod ../go.sum ./

# Baixe as dependências
RUN go mod download

# Copie o resto dos arquivos do projeto para o diretório de trabalho
COPY ../ ./

# Execute o build da aplicação
RUN go build -o nonodo

# Exponha a porta em que a aplicação irá rodar
EXPOSE 8080

# Comando para rodar a aplicação
CMD ["./nonodo", "--http-address=0.0.0.0", "--high-level-graphql", "--enable-debug", "--node-version", "v2", "--load-test-mode", "--db-implementation", "postgres"]