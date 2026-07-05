# Use the official PostGraphile image
FROM graphile/postgraphile:latest

# Expose the API port
EXPOSE 8080

# Copy our configuration file into the container.
# PostGraphile's internal cosmiconfig automatically detects this file on startup.
COPY .postgraphilerc.js .

# We clear out the CMD array so the container boots up using only the default 
# entrypoint and perfectly inherits our .postgraphilerc.js options!
CMD []