import express from 'express';

const app = express();
const port = 3002;

app.get('/api/helios/database/info/:componentName', async (req, res) => {
  const { componentName } = req.params;
  res.status(501).json({
    error: `Deprecated database server endpoint for component=${componentName}`,
  });
});

app.listen(port, () => {
  console.log(`Database info server running on port ${port}`);
});
