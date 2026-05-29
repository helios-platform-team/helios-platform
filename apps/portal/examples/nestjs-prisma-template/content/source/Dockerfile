# ---- Build Stage ----
FROM node:22-alpine3.23 AS builder

WORKDIR /app

# Install dependencies
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
FROM node:22-alpine3.23 AS production

WORKDIR /app

# Install production dependencies only
COPY package*.json ./
RUN npm ci --omit=dev --ignore-scripts --no-audit --no-fund --progress=false

# Copy Prisma schema, config, and generated client from builder
COPY --from=builder /app/prisma ./prisma/
COPY --from=builder /app/prisma.config.ts ./
COPY --from=builder /app/node_modules/.prisma ./node_modules/.prisma/
COPY --from=builder /app/node_modules/@prisma ./node_modules/@prisma/

# Copy built application
COPY --from=builder /app/dist ./dist/

EXPOSE ${{ values.port }}

# Run migrations and start the application
CMD ["npm", "run", "start:migrate:prod"]
