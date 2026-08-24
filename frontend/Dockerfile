# Frontend image. See specs/architecture.md's Docker Test Environment section.

# Dev stage: used by docker-compose.dev.yaml, which bind-mounts ./frontend over /app so edits are
# picked up on browser reload without rebuilding this image. No `COPY . .` here — the bind mount
# supplies the source at runtime; this stage only needs node_modules pre-installed.
FROM node:22-alpine AS dev
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci
EXPOSE 3000
CMD ["npm", "run", "dev", "--", "--host", "0.0.0.0"]

FROM node:22-alpine AS build
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
RUN npm run build

FROM node:22-alpine
RUN npm install -g serve
COPY --from=build /app/dist /app/dist
EXPOSE 3000
CMD ["serve", "-s", "/app/dist", "-l", "3000"]
