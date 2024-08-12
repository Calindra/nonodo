# syntax=docker/dockerfile:1
FROM node:20.16.0-bookworm

WORKDIR /usr/share/app
COPY package.json package-lock.json ./
RUN npm ci

EXPOSE 5001

CMD ["npm", "start"]

