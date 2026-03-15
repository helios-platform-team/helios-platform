# [Impl] Automated Secret Injection #39

## Operator — Secret Injection into Backend Deployment

- [x] Research codebase and understand existing patterns
- [x] Implement [reconcileDatabaseSecretInjection](file:///home/phuochoan/Workspace/HCMUS/4th_Year/Capstone_Projects/helios-platform/apps/operator/internal/controller/database_resources.go#497-552) in Go operator
  - [x] Add new phase (0.9) to inject DB_HOST, DB_USER, DB_PASS into backend Deployment env block
  - [x] Patch the Deployment rendered by CUE/GitOps with env secretKeyRef
- [x] Add comprehensive tests for secret injection logic
  - [x] Unit tests for injection function (3 tests)
  - [x] Integration test with fake client (3 tests)

## Node.js Template — NestJS + Prisma ORM

- [x] Create NestJS + Prisma template directory structure
  - [x] [package.json](file:///home/phuochoan/Workspace/HCMUS/4th_Year/Capstone_Projects/helios-platform/apps/portal/examples/advanced-template/content/source/package.json) with latest NestJS and Prisma deps
  - [x] [prisma/schema.prisma](file:///home/phuochoan/Workspace/HCMUS/4th_Year/Capstone_Projects/helios-platform/apps/portal/examples/nestjs-prisma-template/content/source/prisma/schema.prisma) connecting via DB_HOST, DB_USER, DB_PASS envs
  - [x] `prisma/migrations/` with initial migration
  - [x] NestJS main app module with Prisma service
  - [x] [Dockerfile](file:///home/phuochoan/Workspace/HCMUS/4th_Year/Capstone_Projects/helios-platform/apps/operator/Dockerfile) for containerized deployment
- [x] Update Backstage template for database-backed services

## Verification

- [/] `go build ./...` — compiles cleanly
- [x] `go vet ./...` — no issues
- [x] `make test` — all tests pass
- [/] `make build` — operator binary builds
