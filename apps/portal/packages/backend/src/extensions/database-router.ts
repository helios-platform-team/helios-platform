import { Router } from 'express';

function createDatabaseRouter(): Router {
  const router = Router();

  router.get('/api/helios/database/info/:componentName', (req, res) => {
    const { componentName } = req.params;
    res.status(501).json({
      error: `Deprecated router endpoint for component=${componentName}`,
    });
  });

  router.get('/api/helios/database/health', (_req, res) => {
    res.json({ status: 'ok', service: 'database-api' });
  });

  return router;
}

export default createDatabaseRouter;
