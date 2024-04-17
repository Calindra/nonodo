import { ApolloServer } from "apollo-server"
import { ApolloGateway, IntrospectAndCompose } from "@apollo/gateway"

const gateway = new ApolloGateway({
    supergraphSdl: new IntrospectAndCompose({
        subgraphs: [
            { name: "reader", url: "http://127.0.0.1:8080/graphql" },
            // { name: "example", url: "https://beta.pokeapi.co/graphql/v1beta" },
        ]
    }),
});

const server = new ApolloServer({
    gateway,
    subscriptions: false,
});

server.listen().then(({ url }) => {
    console.log(`🚀 Server ready at ${url}`);
});