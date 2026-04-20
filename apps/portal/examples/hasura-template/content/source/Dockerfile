# Build a custom Hasura image
FROM hasura/graphql-engine:v2.33.0

# Optional: If you want to bake Hasura migrations or metadata into the image, 
# you can copy them here (similar to the PostgREST template copying schema)
# COPY migrations/ /hasura-migrations/
# COPY metadata/ /hasura-metadata/

# Health check to ensure Hasura is running
HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:8080/healthz || exit 1

EXPOSE 8080

# The default entrypoint for the Hasura graphql-engine image handles the start