FROM node:latest

RUN npm install -g postgraphile @graphile-contrib/pg-simplify-inflector postgraphile-plugin-connection-filter

EXPOSE 5000

CMD ["postgraphile", "-c", "postgres://myuser:mypassword@postgres/postgres", "--watch", "--enhance-graphiql", "--dynamic-json", "--append-plugins", "@graphile-contrib/pg-simplify-inflector,postgraphile-plugin-connection-filter", "-n", "0.0.0.0"]
