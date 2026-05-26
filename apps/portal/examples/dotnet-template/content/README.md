# .NET Web API Backstage Template

This template scaffolds an ASP.NET Core Web API that is ready for Helios GitOps deployment.

Generated source repository includes:

- ASP.NET Core Web API project targeting .NET 10 (runtime 10.0.5, SDK 10.0.202)
- Multi-stage Dockerfile
- docker-compose setup for local development with PostgreSQL 18.3
- Environment-driven database configuration using `DB_HOST`, `DB_USER`, `DB_PASSWORD`, and `DB_NAME`

Generated GitOps repository includes:

- HeliosApp manifest with web-service + database trait
