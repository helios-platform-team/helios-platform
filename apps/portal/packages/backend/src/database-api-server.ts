import express from 'express';

const app = express();
const PORT = process.env.DATABASE_API_PORT || 3001;

// Middleware
app.use(express.json());

// Health check endpoint
app.get('/health', (_req, res) => {
  res.json({ status: 'ok', service: 'database-api' });
});

app.get('/api/helios/database/info/:componentName', (req, res) => {
  res.status(501).json({
    error: `Standalone database API server is deprecated, component=${req.params.componentName}`,
  });
});

// Error handling middleware
app.use(
  (
    err: any,
    _req: express.Request,
    res: express.Response,
    _next: express.NextFunction,
  ) => {
  console.error('Error:', err);
  res.status(err.status || 500).json({
    error: err.message || 'Internal server error',
  });
  },
);

app.listen(PORT, () => {
  console.log(`Database API server listening on port ${PORT}`);
  console.log(`Health check at http://localhost:${PORT}/health`);
  console.log(`Database endpoint at http://localhost:${PORT}/api/helios/database/info/:componentName`);
});
