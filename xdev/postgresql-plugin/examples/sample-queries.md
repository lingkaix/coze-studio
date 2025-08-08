# Sample PostgreSQL Queries

Here are some example queries you can test with the PostgreSQL plugin:

## Basic SELECT Queries

### Get all users
```json
{
  "database_url": "postgresql://postgres:password@localhost:5432/myapp",
  "query": "SELECT * FROM users LIMIT 10"
}
```

### Count records
```json
{
  "database_url": "postgresql://postgres:password@localhost:5432/myapp",
  "query": "SELECT COUNT(*) as total_users FROM users"
}
```

### Filter with WHERE clause
```json
{
  "database_url": "postgresql://postgres:password@localhost:5432/myapp",
  "query": "SELECT id, name, email FROM users WHERE age > $1",
  "params": [25]
}
```

## INSERT Operations

### Insert a new user
```json
{
  "database_url": "postgresql://postgres:password@localhost:5432/myapp",
  "query": "INSERT INTO users (name, email, age) VALUES ($1, $2, $3) RETURNING id",
  "params": ["Alice Johnson", "alice@example.com", 28]
}
```

### Insert multiple records
```json
{
  "database_url": "postgresql://postgres:password@localhost:5432/myapp",
  "query": "INSERT INTO products (name, price, category) VALUES ($1, $2, $3), ($4, $5, $6)",
  "params": ["Laptop", 999.99, "Electronics", "Mouse", 29.99, "Electronics"]
}
```

## UPDATE Operations

### Update user information
```json
{
  "database_url": "postgresql://postgres:password@localhost:5432/myapp",
  "query": "UPDATE users SET email = $1, updated_at = NOW() WHERE id = $2",
  "params": ["newemail@example.com", 123]
}
```

### Bulk update
```json
{
  "database_url": "postgresql://postgres:password@localhost:5432/myapp",
  "query": "UPDATE products SET price = price * 0.9 WHERE category = $1",
  "params": ["Electronics"]
}
```

## DELETE Operations

### Delete a specific user
```json
{
  "database_url": "postgresql://postgres:password@localhost:5432/myapp",
  "query": "DELETE FROM users WHERE id = $1",
  "params": [123]
}
```

### Delete with conditions
```json
{
  "database_url": "postgresql://postgres:password@localhost:5432/myapp",
  "query": "DELETE FROM orders WHERE status = $1 AND created_at < $2",
  "params": ["cancelled", "2024-01-01"]
}
```

## Advanced Queries

### JOIN operations
```json
{
  "database_url": "postgresql://postgres:password@localhost:5432/myapp",
  "query": "SELECT u.name, o.total, o.created_at FROM users u JOIN orders o ON u.id = o.user_id WHERE u.id = $1",
  "params": [123]
}
```

### Aggregation with GROUP BY
```json
{
  "database_url": "postgresql://postgres:password@localhost:5432/myapp",
  "query": "SELECT category, COUNT(*) as product_count, AVG(price) as avg_price FROM products GROUP BY category"
}
```

### Complex filtering
```json
{
  "database_url": "postgresql://postgres:password@localhost:5432/myapp",
  "query": "SELECT * FROM orders WHERE total BETWEEN $1 AND $2 AND status IN ($3, $4) ORDER BY created_at DESC",
  "params": [100, 1000, "pending", "shipped"]
}
```

## Schema Operations

### Create a new table
```json
{
  "database_url": "postgresql://postgres:password@localhost:5432/myapp",
  "query": "CREATE TABLE IF NOT EXISTS todos (id SERIAL PRIMARY KEY, title TEXT NOT NULL, completed BOOLEAN DEFAULT FALSE, created_at TIMESTAMP DEFAULT NOW())"
}
```

### Add an index
```json
{
  "database_url": "postgresql://postgres:password@localhost:5432/myapp",
  "query": "CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_users_email ON users(email)"
}
```

## Database Information

### Get all tables and their structure
```json
{
  "database_url": "postgresql://postgres:password@localhost:5432/myapp"
}
```

## Analytics Queries

### Monthly sales report
```json
{
  "database_url": "postgresql://postgres:password@localhost:5432/myapp",
  "query": "SELECT DATE_TRUNC('month', created_at) as month, SUM(total) as total_sales, COUNT(*) as order_count FROM orders WHERE created_at >= $1 GROUP BY month ORDER BY month",
  "params": ["2024-01-01"]
}
```

### Top customers
```json
{
  "database_url": "postgresql://postgres:password@localhost:5432/myapp",
  "query": "SELECT u.name, u.email, COUNT(o.id) as order_count, SUM(o.total) as total_spent FROM users u JOIN orders o ON u.id = o.user_id GROUP BY u.id, u.name, u.email ORDER BY total_spent DESC LIMIT $1",
  "params": [10]
}
```

## Tips for Using the Plugin

1. **Always use parameterized queries** with `$1, $2, $3...` placeholders instead of string concatenation
2. **Test queries with LIMIT** first to avoid returning too much data
3. **Use transactions** for multiple related operations
4. **Check execution time** in the response to optimize slow queries
5. **Use the database info endpoint** to explore the schema before writing queries