# Build a custom PostgREST image with your database schema and configuration
# This extends the official postgrest/postgrest image with your SQL schema files
FROM postgrest/postgrest:v12.2.0

# Copy your database schema files
# Place .sql files in the schema/ directory that define your database tables
# Example: schema/01-tables.sql, schema/02-permissions.sql
# These will be loaded into PostgreSQL when the database starts
COPY schema/ /schema/

# Copy your PostgREST configuration
# Includes settings like port, database schema to expose, JWT authentication, etc.
COPY postgrestrc.conf /etc/postgrest/postgrestrc.conf

# Health check to ensure PostgREST is running and ready
HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:3000/ || exit 1

# Expose the PostgREST API port
EXPOSE 3000

# Start PostgREST with your custom configuration
CMD ["postgrest", "/etc/postgrest/postgrestrc.conf"]
