import { Router, Request, Response } from 'express';
import { DatabaseService } from './service';

export function createDatabaseRouter(): Router {
  const router = Router();
  let databaseService: DatabaseService;

  // Initialize service on first request
  const getService = () => {
    if (!databaseService) {
      databaseService = new DatabaseService();
    }
    return databaseService;
  };

  /**
   * GET /database/info/:componentName
   * Fetch database connectivity information for a component
   */
  router.get('/info/:componentName', async (req: Request, res: Response) => {
    try {
      const { componentName } = req.params;

      if (!componentName) {
        return res.status(400).json({ error: 'Component name is required' });
      }

      const service = getService();
      const databaseInfo = await service.getDatabaseInfo(componentName);
      return res.json(databaseInfo);
    } catch (error) {
      console.error('Database info fetch error:', error);
      const errorMessage =
        error instanceof Error ? error.message : 'Unknown error';
      return res.status(500).json({ error: errorMessage });
    }
  });

  return router;
}
