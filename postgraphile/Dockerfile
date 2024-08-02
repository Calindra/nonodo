# syntax=docker/dockerfile:1
FROM node:20.16.0-bookworm

WORKDIR /usr/share/app
COPY package.json package-lock.json ./
RUN npm ci

# RUN npm install -g postgraphile @graphile-contrib/pg-simplify-inflector postgraphile-plugin-connection-filter

EXPOSE 5000
# @graphile/pg-pubsub,

# CMD ["npx", "postgraphile", "-c", "postgres://myuser:mypassword@postgres/mydatabase", "--watch", "--enhance-graphiql", "--dynamic-json", "--append-plugins", "@graphile-contrib/pg-simplify-inflector,postgraphile-plugin-connection-filter", "-n", "0.0.0.0"]
CMD ["npm", "start"]

