export interface DatabaseConfig {
  host: string;
  port: number;
  database: string;
  username: string;
  password: string;
}

export interface QueryRequest {
  database_url: string;
  query: string;
  params?: any[];
}

export interface QueryResponse {
  success: boolean;
  data?: any[];
  error?: string;
  affected_rows?: number;
  execution_time_ms?: number;
}

export interface DatabaseInfoRequest {
  database_url: string;
}

export interface DatabaseInfoResponse {
  success: boolean;
  tables?: TableInfo[];
  error?: string;
}

export interface TableInfo {
  table_name: string;
  columns: ColumnInfo[];
}

export interface ColumnInfo {
  column_name: string;
  data_type: string;
  is_nullable: boolean;
  column_default?: string;
}

export interface ErrorResponse {
  success: false;
  error: string;
}