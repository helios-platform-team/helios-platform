# ---- Build Stage ----
FROM node:24.16.0-alpine3.23 AS builder

WORKDIR /app

# Install dependencies (maximizes layer caching)
COPY package*.json ./
RUN npm ci --ignore-scripts --no-audit --no-fund --progress=false

# Copy source and build
COPY . .
RUN npm run build

# ---- Production Stage ----
FROM nginx:1.31.2-alpine3.23-slim AS production

# Copy built application to Nginx server
COPY --from=builder /app/dist /usr/share/nginx/html

# Configure port dynamically at runtime/build-time
ARG PORT=${{ values.port }}
ENV PORT=${PORT}

RUN echo 'server { \
    listen '${PORT}'; \
    \
    # Serve static React files \
    location / { \
        root /usr/share/nginx/html; \
        index index.html index.htm; \
        try_files $uri $uri/ /index.html; \
    } \
    \
    # Proxy /api/ requests to the internal K8s backend \
    location /api/ { \
        proxy_pass http://${{ values.backendComponent }}.default.svc.cluster.local:${{ values.backendPort }}/; \
        proxy_http_version 1.1; \
        proxy_set_header Host $host; \
        proxy_set_header X-Real-IP $remote_addr; \
    } \
}' > /etc/nginx/conf.d/default.conf

# Labels for metadata
LABEL org.opencontainers.image.source="https://github.com/helios-platform-team/helios-platform" \
  org.opencontainers.image.description="Helios Multi-Option React Template Application" \
  org.opencontainers.image.licenses="MIT"

EXPOSE ${PORT}

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:${PORT}/ || exit 1

CMD ["nginx", "-g", "daemon off;"]
