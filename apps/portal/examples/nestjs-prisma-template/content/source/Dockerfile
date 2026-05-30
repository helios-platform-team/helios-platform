# ---- Build Stage ----
FROM node:24.16.0-alpine3.23 AS builder

WORKDIR /app

# Install dependencies (maximizes layer caching)
COPY package*.json ./
RUN npm ci --ignore-scripts --no-audit --no-fund --progress=false

# Copy Prisma schema and config, then generate client
COPY prisma ./prisma/
COPY prisma.config.ts ./
RUN npx prisma generate

# Build NestJS application
COPY . .
RUN npm run build

# ---- Production Stage ----
FROM node:24.16.0-alpine3.23 AS production

# Security: Create non-root user and group
RUN addgroup -g 1001 -S nodejs && \
  adduser -S nestjs -u 1001 -G nodejs

WORKDIR /app

# Copy package files
COPY --chown=nestjs:nodejs package*.json ./

# Install production dependencies only
RUN npm ci --omit=dev --ignore-scripts --no-audit --no-fund --progress=false

# Copy Prisma schema, config, and generated client from builder
COPY --from=builder --chown=nestjs:nodejs /app/prisma ./prisma/
COPY --from=builder --chown=nestjs:nodejs /app/prisma.config.ts ./
COPY --from=builder --chown=nestjs:nodejs /app/node_modules/.prisma ./node_modules/.prisma/
COPY --from=builder --chown=nestjs:nodejs /app/node_modules/@prisma ./node_modules/@prisma/

# Copy built application
COPY --from=builder --chown=nestjs:nodejs /app/dist ./dist/

# Set environment variables
ENV NODE_ENV=production

# Labels for metadata
LABEL org.opencontainers.image.source="https://github.com/helios-platform-team/helios-platform" \
  org.opencontainers.image.description="Helios NestJS Prisma Template Application" \
  org.opencontainers.image.licenses="MIT"

# Security: Switch to non-root user
USER nestjs

EXPOSE ${{ values.port }}

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD node -e "require('http').get('http://localhost:' + (process.env.PORT || '${{ values.port }}') + '/health', (r) => process.exit(r.statusCode === 200 ? 0 : 1))"

# Run migrations and start the application
CMD ["npm", "run", "start:migrate:prod"]
