import postgres from 'postgres';
import type { 
  QueryRequest, 
  QueryResponse, 
  DatabaseInfoRequest, 
  DatabaseInfoResponse,
  TableInfo,
  ColumnInfo
} from './types';

export class DatabaseService {
  private connections = new Map<string, ReturnType<typeof postgres>>();

  private getConnection(databaseUrl: string) {
    if (!this.connections.has(databaseUrl)) {
      const sql = postgres(databaseUrl, {
        max: 1, // Single connection per URL to avoid connection pool issues
        idle_timeout: 20,
        connect_timeout: 10,
      });
      this.connections.set(databaseUrl, sql);
    }
    return this.connections.get(databaseUrl)!;
  }

  async executeQuery(request: QueryRequest): Promise<QueryResponse> {
    const startTime = Date.now();
    
    try {
      // Validate the database URL
      if (!request.database_url || !request.database_url.startsWith('postgresql://')) {
        return {
          success: false,
          error: 'Invalid database URL. Must be a PostgreSQL connection string.'
        };
      }

      // Validate the query
      if (!request.query || request.query.trim().length === 0) {
        return {
          success: false,
          error: 'Query cannot be empty.'
        };
      }

      const sql = this.getConnection(request.database_url);
      const params = request.params || [];
      
      // Execute the query
      const result = await sql.unsafe(request.query, params);
      const executionTime = Date.now() - startTime;

      // Handle different types of queries
      if (Array.isArray(result)) {
        return {
          success: true,
          data: result,
          execution_time_ms: executionTime
        };
      } else {
        // For INSERT, UPDATE, DELETE operations
        return {
          success: true,
          affected_rows: result.count || 0,
          execution_time_ms: executionTime
        };
      }
    } catch (error: any) {
      const executionTime = Date.now() - startTime;
      
      return {
        success: false,
        error: error.message || 'Database query failed',
        execution_time_ms: executionTime
      };
    }
  }

  async getDatabaseInfo(request: DatabaseInfoRequest): Promise<DatabaseInfoResponse> {
    try {
      if (!request.database_url || !request.database_url.startsWith('postgresql://')) {
        return {
          success: false,
          error: 'Invalid database URL. Must be a PostgreSQL connection string.'
        };
      }

      const sql = this.getConnection(request.database_url);
      
      // Get all tables in the public schema
      const tablesResult = await sql`
        SELECT table_name 
        FROM information_schema.tables 
        WHERE table_schema = 'public' AND table_type = 'BASE TABLE'
        ORDER BY table_name
      `;

      const tables: TableInfo[] = [];
      
      for (const table of tablesResult) {
        // Get column information for each table
        const columnsResult = await sql`
          SELECT 
            column_name,
            data_type,
            is_nullable,
            column_default
          FROM information_schema.columns 
          WHERE table_schema = 'public' AND table_name = ${table.table_name}
          ORDER BY ordinal_position
        `;

        const columns: ColumnInfo[] = columnsResult.map((col: any) => ({
          column_name: col.column_name,
          data_type: col.data_type,
          is_nullable: col.is_nullable === 'YES',
          column_default: col.column_default || undefined
        }));

        tables.push({
          table_name: table.table_name,
          columns
        });
      }

      return {
        success: true,
        tables
      };
    } catch (error: any) {
      return {
        success: false,
        error: error.message || 'Failed to retrieve database information'
      };
    }
  }

  async closeConnection(databaseUrl: string) {
    const connection = this.connections.get(databaseUrl);
    if (connection) {
      await connection.end();
      this.connections.delete(databaseUrl);
    }
  }

  async closeAllConnections() {
    for (const [url, connection] of this.connections.entries()) {
      await connection.end();
    }
    this.connections.clear();
  }
}