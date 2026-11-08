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
RUN npm run api:generate
EXPOSE 8081
HEALTHCHECK --interval=10s --timeout=5s --start-period=45s --retries=3 \
  CMD node -e "fetch('http://127.0.0.1:8081/_expo/open').then(r => process.exit(r.ok ? 0 : 1)).catch(() => process.exit(1))"
CMD ["npx", "expo", "start", "--host", "lan", "--port", "8081", "--web"]
