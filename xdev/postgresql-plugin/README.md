# PostgreSQL Database Plugin for Coze Studio

A powerful plugin that enables Coze Studio to connect to PostgreSQL databases, execute queries, and retrieve database schema information. Built with Bun and TypeScript for high performance and type safety.

## Features

- 🗄️ **Execute SQL Queries**: Run SELECT, INSERT, UPDATE, DELETE operations
- 🔍 **Database Introspection**: Get table and column information
- 🛡️ **Parameterized Queries**: Secure query execution with parameter binding
- 🔐 **API Key Authentication**: Optional security layer
- ⚡ **High Performance**: Built with Bun for fast execution
- 🔧 **TypeScript**: Full type safety and excellent developer experience

## Quick Start

### 1. Installation

```bash
cd postgresql-plugin
bun install
```

### 2. Configuration

Copy the environment template and configure your settings:

```bash
cp .env.example .env
```

Edit `.env` with your configuration:

```bash
# Server Configuration
PORT=3000
HOST=localhost

# Database Configuration - This is just an example, actual connections are per-request
DATABASE_URL=postgresql://username:password@localhost:5432/database_name

# Security (optional)
API_KEY=your-secure-api-key-here

# Coze Studio Configuration
COZE_API_URL=http://localhost:8080
SPACE_ID=your-space-id
PROJECT_ID=your-project-id  # optional
```

### 3. Start the Plugin Server

```bash
# Development mode with hot reload
bun run dev

# Production mode
bun run build
bun run start
```

### 4. Register with Coze Studio

```bash
bun run register
```

## API Endpoints

### Health Check
```http
GET /health
```

Returns the service status and timestamp.

### Execute Query
```http
POST /query
Authorization: Bearer your-api-key
Content-Type: application/json

{
  "database_url": "postgresql://user:pass@localhost:5432/db",
  "query": "SELECT * FROM users WHERE age > $1",
  "params": [25]
}
```

### Get Database Info
```http
POST /database-info
Authorization: Bearer your-api-key
Content-Type: application/json

{
  "database_url": "postgresql://user:pass@localhost:5432/db"
}
```

## Usage Examples

### Basic SELECT Query
```json
{
  "database_url": "postgresql://postgres:password@localhost:5432/myapp",
  "query": "SELECT id, name, email FROM users LIMIT 10"
}
```

### Parameterized Query
```json
{
  "database_url": "postgresql://postgres:password@localhost:5432/myapp",
  "query": "INSERT INTO users (name, email, age) VALUES ($1, $2, $3)",
  "params": ["John Doe", "john@example.com", 30]
}
```

### Complex Query with Multiple Parameters
```json
{
  "database_url": "postgresql://postgres:password@localhost:5432/myapp",
  "query": "SELECT * FROM orders WHERE created_at BETWEEN $1 AND $2 AND status = $3",
  "params": ["2024-01-01", "2024-12-31", "completed"]
}
```

### Database Schema Exploration
```json
{
  "database_url": "postgresql://postgres:password@localhost:5432/myapp"
}
```

## Security Considerations

- **API Key Authentication**: Set `API_KEY` environment variable to enable authentication
- **Parameterized Queries**: Always use `$1, $2, $3...` placeholders instead of string concatenation
- **Connection Management**: Each database URL gets its own connection pool
- **Input Validation**: All inputs are validated before processing

## Error Handling

The plugin provides detailed error messages for common issues:

- Invalid database URLs
- Connection failures
- SQL syntax errors
- Authentication failures
- Parameter mismatches

Example error response:
```json
{
  "success": false,
  "error": "relation \"nonexistent_table\" does not exist",
  "execution_time_ms": 12
}
```

## Development

### Project Structure
```
postgresql-plugin/
├── src/
│   ├── index.ts          # Main server
│   ├── database.ts       # Database service
│   ├── types.ts          # TypeScript definitions
│   └── register.ts       # Plugin registration
├── openapi.yaml          # API specification
├── ai_plugin.json        # Plugin manifest
└── package.json
```

### Building
```bash
bun run build
```

### Testing
```bash
bun test
```

## Plugin Registration Details

The plugin uses Coze Studio's RegisterPlugin API with these components:

- **AI Plugin Manifest** (`ai_plugin.json`): Defines the plugin metadata
- **OpenAPI Specification** (`openapi.yaml`): Describes the API endpoints
- **Service Token**: Optional authentication token for the plugin

## Troubleshooting

### Connection Issues
- Verify PostgreSQL is running
- Check database URL format: `postgresql://user:password@host:port/database`
- Ensure network connectivity

### Registration Issues
- Verify `SPACE_ID` is correct
- Check that Coze Studio is running
- Validate JSON and YAML syntax

### Authentication Issues
- Ensure API key is set correctly
- Check Bearer token format in requests

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## License

This project is licensed under the Apache License 2.0 - see the LICENSE file for details.