# ---- Build Stage ----
FROM node:24-alpine AS builder

WORKDIR /app

# Install dependencies
COPY package*.json ./
RUN npm ci

# Copy Prisma schema and config, then generate client
COPY prisma ./prisma/
COPY prisma.config.ts ./
RUN npx prisma generate

# Build NestJS application
COPY . .
RUN npm run build

# ---- Production Stage ----
FROM node:24-alpine AS production

WORKDIR /app

# Install production dependencies only
COPY package*.json ./
RUN npm ci --omit=dev

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
