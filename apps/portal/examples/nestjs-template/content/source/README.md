# ${{ values.name }} (NestJS Application)

This project is scaffolded from the Helios Backstage NestJS template.

Pinned versions:

- Node.js: 24.16.0
{% if values.hasDatabase -%}
- Prisma Client: 7.8.0
- PostgreSQL: 18.4 (via Database trait / Helios Operator)
{%- endif %}

## Local Development

1. Install dependencies:

```bash
npm install
```

{% if values.hasDatabase -%}
2. Generate Prisma Client:

```bash
npm run prisma:generate
```

{%- endif %}

1. Start application in development mode:

```bash
npm run start:dev
```

1. Check health endpoint:

<http://localhost:${{ values.port }}/health>
