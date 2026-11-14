# Development server only. Native Android/iOS apps run on phones or emulators.
FROM node:24-bookworm-slim
WORKDIR /app
RUN chown node:node /app
USER node
RUN mkdir -p /home/node/.expo
COPY --chown=node:node app/package.json app/package-lock.json ./
RUN npm ci

COPY --chown=node:node app/ .
COPY --chown=node:node backend/api/ /backend/api/

EXPOSE 8081

CMD ["npx", "expo", "start", "--host", "lan", "--port", "8081", "--web"]
