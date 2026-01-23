const express = require('express');
const app = express();
// Use the port from environment or default to the template value
const port = process.env.PORT || ${{ values.port }};

app.get('/', (req, res) => {
  res.send('Hello from Helios! App Name: ${{ values.appName }}');
});

app.listen(port, () => {
  console.log(`Helios app listening on port ${port}`);
});