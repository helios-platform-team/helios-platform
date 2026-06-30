# ---- Build Stage ----
FROM node:24.18.0-alpine3.23 AS builder

WORKDIR /app

# Install all dependencies (dev + prod)
COPY package*.json ./
RUN npm ci --no-audit --no-fund --progress=false

{%- if values.hasDatabase %}
# Copy Prisma files and generate Prisma Client
COPY prisma ./prisma/
COPY prisma.config.js ./
RUN npx prisma generate
{%- endif %}

# ---- Production Stage ----
FROM node:24.18.0-alpine3.23 AS production

# Security: Create non-root user and group
RUN addgroup -g 1001 -S nodejs && \
  adduser -S nodejs -u 1001 -G nodejs

WORKDIR /app

# Copy package files
COPY --chown=nodejs:nodejs package*.json ./

# Install production dependencies only (using npm ci --omit=dev)
RUN npm ci --omit=dev --ignore-scripts --no-audit --no-fund --progress=false

{%- if values.hasDatabase %}
# Copy Prisma files and configuration from builder, then generate client
COPY --from=builder --chown=nodejs:nodejs /app/prisma ./prisma/
COPY --from=builder --chown=nodejs:nodejs /app/prisma.config.js ./
RUN npx prisma generate
{%- endif %}

# Copy application files (index.js)
COPY --chown=nodejs:nodejs index.js ./
{%- if values.hasDatabase %}
COPY --chown=nodejs:nodejs prisma.service.js ./
{%- endif %}

# Set environment variables
ENV NODE_ENV=production

# Labels for metadata
LABEL org.opencontainers.image.source="https://github.com/helios-platform-team/helios-platform" \
  org.opencontainers.image.description="Helios Node.js Prisma Template Application" \
  org.opencontainers.image.licenses="MIT"

# Security: Switch to non-root user
USER nodejs

EXPOSE ${{ values.port }}

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD node -e "require('http').get('http://localhost:' + (process.env.PORT || '${{ values.port }}') + '/health', (r) => process.exit(r.statusCode === 200 ? 0 : 1))"

{%- if values.hasDatabase %}
# Run migrations and start the application
CMD ["npm", "run", "start:prod"]
{%- else %}
# Start the application
CMD ["node", "index.js"]
{%- endif %}
