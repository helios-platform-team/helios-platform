const http = require('http');
const port = ${{ values.port }};
{% if values.hasDatabase -%}
const { PrismaService } = require('./prisma.service');
const prisma = new PrismaService();
{%- endif %}

const server = http.createServer(async (req, res) => {
    // Add CORS headers for frontend integration
    res.setHeader('Access-Control-Allow-Origin', '*');
    res.setHeader('Access-Control-Allow-Methods', 'GET, POST, PUT, DELETE, OPTIONS');
    res.setHeader('Access-Control-Allow-Headers', 'Content-Type, Authorization');

    // Handle preflight request
    if (req.method === 'OPTIONS') {
        res.writeHead(204);
        res.end();
        return;
    }

    if (req.url === '/' && req.method === 'GET') {
        res.writeHead(200, { 'Content-Type': 'text/plain' });
        res.end('Hello World! This is ${{ values.name }}');
        return;
    }

    if (req.url === '/health' && req.method === 'GET') {
        {% if values.hasDatabase -%}
        try {
            await prisma.$queryRaw`SELECT 1`;
            res.writeHead(200, { 'Content-Type': 'application/json' });
            res.end(JSON.stringify({ status: 'ok', database: 'connected' }));
        } catch (err) {
            res.writeHead(500, { 'Content-Type': 'application/json' });
            res.end(JSON.stringify({ status: 'error', database: `disconnected: ${err.message}` }));
        }
        {%- else -%}
        res.writeHead(200, { 'Content-Type': 'application/json' });
        res.end(JSON.stringify({ status: 'ok' }));
        {%- endif %}
        return;
    }

    res.writeHead(404, { 'Content-Type': 'text/plain' });
    res.end('Not Found');
});

// For clean shutdown (closing the database pool and server)
async function gracefulShutdown() {
    console.log('Shutting down server...');
    server.close(async () => {
        {% if values.hasDatabase -%}
        try {
            await prisma.onModuleDestroy();
            console.log('Database connection closed.');
        } catch (err) {
            console.error('Error closing database connection:', err);
        }
        {%- endif %}
        process.exit(0);
    });
}

process.on('SIGTERM', gracefulShutdown);
process.on('SIGINT', gracefulShutdown);

// Initialize DB and start server
async function start() {
    {% if values.hasDatabase -%}
    await prisma.onModuleInit();
    {%- endif %}
    server.listen(port, () => {
        console.log(`Example app listening at http://localhost:${port}`);
    });
}

start();
