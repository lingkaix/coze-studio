import { DatabaseService } from './database';
import type { QueryRequest, DatabaseInfoRequest } from './types';

const database = new DatabaseService();

// Get configuration from environment
const PORT = process.env.PORT ? parseInt(process.env.PORT) : 3000;
const HOST = process.env.HOST || '0.0.0.0';
const API_KEY = process.env.API_KEY;

// Simple API key authentication middleware
function authenticateRequest(req: Request): boolean {
  if (!API_KEY) {
    return true; // No authentication required if API_KEY is not set
  }
  
  const authHeader = req.headers.get('Authorization');
  const providedKey = authHeader?.replace('Bearer ', '');
  return providedKey === API_KEY;
}

async function handleQuery(req: Request): Promise<Response> {
  try {
    if (!authenticateRequest(req)) {
      return new Response(JSON.stringify({ success: false, error: 'Unauthorized' }), {
        status: 401,
        headers: { 'Content-Type': 'application/json' }
      });
    }

    const queryRequest: QueryRequest = await req.json();
    const result = await database.executeQuery(queryRequest);
    
    return new Response(JSON.stringify(result), {
      status: result.success ? 200 : 400,
      headers: { 'Content-Type': 'application/json' }
    });
  } catch (error) {
    return new Response(JSON.stringify({ 
      success: false, 
      error: 'Invalid request body' 
    }), {
      status: 400,
      headers: { 'Content-Type': 'application/json' }
    });
  }
}

async function handleDatabaseInfo(req: Request): Promise<Response> {
  try {
    if (!authenticateRequest(req)) {
      return new Response(JSON.stringify({ success: false, error: 'Unauthorized' }), {
        status: 401,
        headers: { 'Content-Type': 'application/json' }
      });
    }

    const infoRequest: DatabaseInfoRequest = await req.json();
    const result = await database.getDatabaseInfo(infoRequest);
    
    return new Response(JSON.stringify(result), {
      status: result.success ? 200 : 400,
      headers: { 'Content-Type': 'application/json' }
    });
  } catch (error) {
    return new Response(JSON.stringify({ 
      success: false, 
      error: 'Invalid request body' 
    }), {
      status: 400,
      headers: { 'Content-Type': 'application/json' }
    });
  }
}

function handleHealthCheck(): Response {
  return new Response(JSON.stringify({ 
    status: 'healthy',
    timestamp: new Date().toISOString(),
    service: 'postgresql-plugin'
  }), {
    headers: { 'Content-Type': 'application/json' }
  });
}

const server = Bun.serve({
  port: PORT,
  hostname: HOST,
  async fetch(req) {
    const url = new URL(req.url);
    
    // CORS headers
    const corsHeaders = {
      'Access-Control-Allow-Origin': '*',
      'Access-Control-Allow-Methods': 'GET, POST, OPTIONS',
      'Access-Control-Allow-Headers': 'Content-Type, Authorization',
    };

    // Handle preflight requests
    if (req.method === 'OPTIONS') {
      return new Response(null, { status: 200, headers: corsHeaders });
    }

    // Add CORS headers to all responses
    const addCorsHeaders = (response: Response) => {
      Object.entries(corsHeaders).forEach(([key, value]) => {
        response.headers.set(key, value);
      });
      return response;
    };

    // Route handling
    switch (url.pathname) {
      case '/health':
        return addCorsHeaders(handleHealthCheck());
      
      case '/query':
        if (req.method === 'POST') {
          return addCorsHeaders(await handleQuery(req));
        }
        break;
      
      case '/database-info':
        if (req.method === 'POST') {
          return addCorsHeaders(await handleDatabaseInfo(req));
        }
        break;
      
      default:
        return addCorsHeaders(new Response(JSON.stringify({ 
          error: 'Not found',
          available_endpoints: ['/health', '/query', '/database-info']
        }), {
          status: 404,
          headers: { 'Content-Type': 'application/json' }
        }));
    }

    return addCorsHeaders(new Response('Method not allowed', { status: 405 }));
  },
});

// Graceful shutdown
process.on('SIGINT', async () => {
  console.log('Shutting down gracefully...');
  await database.closeAllConnections();
  server.stop();
  process.exit(0);
});

console.log(`🚀 PostgreSQL Plugin server running at http://${HOST}:${PORT}`);
console.log(`📚 Available endpoints:`);
console.log(`  GET  /health - Health check`);
console.log(`  POST /query - Execute SQL query`);
console.log(`  POST /database-info - Get database schema information`);

if (API_KEY) {
  console.log(`🔐 API Key authentication enabled`);
} else {
  console.log(`⚠️  No API Key set - authentication disabled`);
}