import express from 'express';
import { createDatabaseRouter } from '@helios/plugin-database-backend';

const app = express();
const PORT = process.env.DATABASE_API_PORT || 3001;

// Middleware
app.use(express.json());

// Health check endpoint
app.get('/health', (req, res) => {
  res.json({ status: 'ok', service: 'database-api' });
});

// Mount database router
const databaseRouter = createDatabaseRouter();
app.use('/api/helios/database', databaseRouter);

// Error handling middleware
app.use((err: any, req: express.Request, res: express.Response, next: express.NextFunction) => {
  console.error('Error:', err);
  res.status(err.status || 500).json({
    error: err.message || 'Internal server error',
  });
});

app.listen(PORT, () => {
  console.log(`Database API server listening on port ${PORT}`);
  console.log(`Health check at http://localhost:${PORT}/health`);
  console.log(`Database endpoint at http://localhost:${PORT}/api/helios/database/info/:componentName`);
});
