const express = require('express');
const app = express();
const port = ${{ values.port }};

app.get('/', (req, res) => {
    res.send('Hello World! This is ${{ values.name }}');
});

app.listen(port, () => {
    console.log(`Example app listening at http://localhost:${port}`);
});
