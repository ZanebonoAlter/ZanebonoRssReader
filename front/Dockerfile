FROM node:22-alpine AS build

WORKDIR /app

ARG NPM_CONFIG_REGISTRY
ARG HTTP_PROXY
ARG HTTPS_PROXY
ARG NO_PROXY

RUN corepack enable

COPY package.json pnpm-lock.yaml ./
RUN if [ -n "$NPM_CONFIG_REGISTRY" ]; then pnpm config set registry "$NPM_CONFIG_REGISTRY"; fi \
  && pnpm install --frozen-lockfile

COPY . .
RUN pnpm build

FROM node:22-alpine

WORKDIR /app

COPY --from=build /app/.output /app/.output

ENV NODE_ENV=production \
  NITRO_HOST=0.0.0.0 \
  NITRO_PORT=3000 \
  HOST=0.0.0.0 \
  PORT=3000

EXPOSE 3000

CMD ["node", ".output/server/index.mjs"]
