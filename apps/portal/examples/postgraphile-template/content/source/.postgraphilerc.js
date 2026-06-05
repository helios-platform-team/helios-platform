module.exports = {
  options: {
    // Node.js will safely read the pre-formatted secret injected by the Operator!
    connection: process.env.POSTGRAPHILE_DB_URI,
    schema: ["public"],
    host: "0.0.0.0",
    port: 5000,
    watch: true,
    enhanceGraphiql: true,
    dynamicJson: true
  }
};