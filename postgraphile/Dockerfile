FROM node:latest

RUN npm install -g postgraphile @graphile-contrib/pg-simplify-inflector postgraphile-plugin-connection-filter @graphile/pg-pubsub

EXPOSE 5000

CMD ["postgraphile", "-c", "postgres://myuser:mypassword@postgres/mydatabase", "--watch", "--enhance-graphiql", "--dynamic-json", "--append-plugins", "@graphile/pg-pubsub,@graphile-contrib/pg-simplify-inflector,postgraphile-plugin-connection-filter", "--subscriptions", "--simple-subscriptions", "-n", "0.0.0.0"]
