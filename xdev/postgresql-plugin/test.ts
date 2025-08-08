#!/usr/bin/env bun

// Simple test script to validate the plugin functionality
// This doesn't require a real PostgreSQL database - just tests the API endpoints

const BASE_URL = 'http://localhost:3000';
const API_KEY = process.env.API_KEY;

interface TestResult {
  name: string;
  passed: boolean;
  error?: string;
}

async function makeRequest(endpoint: string, method: string = 'GET', body?: any): Promise<Response> {
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
  };
  
  if (API_KEY) {
    headers['Authorization'] = `Bearer ${API_KEY}`;
  }

  return await fetch(`${BASE_URL}${endpoint}`, {
    method,
    headers,
    body: body ? JSON.stringify(body) : undefined,
  });
}

async function testHealthCheck(): Promise<TestResult> {
  try {
    const response = await makeRequest('/health');
    const data = await response.json();
    
    if (response.status === 200 && data.status === 'healthy') {
      return { name: 'Health Check', passed: true };
    } else {
      return { name: 'Health Check', passed: false, error: `Unexpected response: ${JSON.stringify(data)}` };
    }
  } catch (error) {
    return { name: 'Health Check', passed: false, error: error.message };
  }
}

async function testInvalidQuery(): Promise<TestResult> {
  try {
    const response = await makeRequest('/query', 'POST', {
      database_url: 'invalid-url',
      query: 'SELECT 1',
    });
    const data = await response.json();
    
    if (response.status === 400 && data.success === false && data.error.includes('Invalid database URL')) {
      return { name: 'Invalid Query Validation', passed: true };
    } else {
      return { name: 'Invalid Query Validation', passed: false, error: `Expected validation error, got: ${JSON.stringify(data)}` };
    }
  } catch (error) {
    return { name: 'Invalid Query Validation', passed: false, error: error.message };
  }
}

async function testEmptyQuery(): Promise<TestResult> {
  try {
    const response = await makeRequest('/query', 'POST', {
      database_url: 'postgresql://user:pass@localhost:5432/db',
      query: '',
    });
    const data = await response.json();
    
    if (response.status === 400 && data.success === false && data.error.includes('Query cannot be empty')) {
      return { name: 'Empty Query Validation', passed: true };
    } else {
      return { name: 'Empty Query Validation', passed: false, error: `Expected validation error, got: ${JSON.stringify(data)}` };
    }
  } catch (error) {
    return { name: 'Empty Query Validation', passed: false, error: error.message };
  }
}

async function testDatabaseInfoValidation(): Promise<TestResult> {
  try {
    const response = await makeRequest('/database-info', 'POST', {
      database_url: 'invalid-url',
    });
    const data = await response.json();
    
    if (response.status === 400 && data.success === false && data.error.includes('Invalid database URL')) {
      return { name: 'Database Info Validation', passed: true };
    } else {
      return { name: 'Database Info Validation', passed: false, error: `Expected validation error, got: ${JSON.stringify(data)}` };
    }
  } catch (error) {
    return { name: 'Database Info Validation', passed: false, error: error.message };
  }
}

async function testNotFoundEndpoint(): Promise<TestResult> {
  try {
    const response = await makeRequest('/nonexistent');
    
    if (response.status === 404) {
      return { name: '404 Handler', passed: true };
    } else {
      return { name: '404 Handler', passed: false, error: `Expected 404, got ${response.status}` };
    }
  } catch (error) {
    return { name: '404 Handler', passed: false, error: error.message };
  }
}

async function testMethodNotAllowed(): Promise<TestResult> {
  try {
    const response = await makeRequest('/query', 'GET');
    
    if (response.status === 405) {
      return { name: 'Method Not Allowed', passed: true };
    } else {
      return { name: 'Method Not Allowed', passed: false, error: `Expected 405, got ${response.status}` };
    }
  } catch (error) {
    return { name: 'Method Not Allowed', passed: false, error: error.message };
  }
}

async function testAuthenticationIfEnabled(): Promise<TestResult> {
  if (!API_KEY) {
    return { name: 'Authentication (Skip - No API Key)', passed: true };
  }

  try {
    // Test without API key
    const response = await fetch(`${BASE_URL}/query`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        database_url: 'postgresql://user:pass@localhost:5432/db',
        query: 'SELECT 1',
      }),
    });
    
    if (response.status === 401) {
      return { name: 'Authentication', passed: true };
    } else {
      return { name: 'Authentication', passed: false, error: `Expected 401, got ${response.status}` };
    }
  } catch (error) {
    return { name: 'Authentication', passed: false, error: error.message };
  }
}

async function runTests() {
  console.log('🧪 Running PostgreSQL Plugin Tests...\n');
  console.log(`📡 Testing against: ${BASE_URL}`);
  console.log(`🔐 API Key: ${API_KEY ? 'Configured' : 'Not configured'}\n`);

  const tests = [
    testHealthCheck,
    testInvalidQuery,
    testEmptyQuery,
    testDatabaseInfoValidation,
    testNotFoundEndpoint,
    testMethodNotAllowed,
    testAuthenticationIfEnabled,
  ];

  const results: TestResult[] = [];

  for (const test of tests) {
    try {
      const result = await test();
      results.push(result);
      
      const status = result.passed ? '✅' : '❌';
      console.log(`${status} ${result.name}`);
      if (!result.passed && result.error) {
        console.log(`   Error: ${result.error}`);
      }
    } catch (error) {
      results.push({ name: test.name, passed: false, error: error.message });
      console.log(`❌ ${test.name}`);
      console.log(`   Error: ${error.message}`);
    }
  }

  const passed = results.filter(r => r.passed).length;
  const total = results.length;

  console.log(`\n📊 Test Results: ${passed}/${total} tests passed`);

  if (passed === total) {
    console.log('🎉 All tests passed! The plugin is working correctly.');
    console.log('\n📋 Next steps:');
    console.log('  1. Set up your PostgreSQL database');
    console.log('  2. Register the plugin with: bun run register');
    console.log('  3. Test with real database queries in Coze Studio');
  } else {
    console.log('⚠️  Some tests failed. Please check the server logs and configuration.');
    process.exit(1);
  }
}

// Check if server is running
async function checkServer() {
  try {
    await fetch(`${BASE_URL}/health`);
  } catch (error) {
    console.error('❌ Cannot connect to the plugin server.');
    console.log('💡 Make sure to start the server first with: bun run dev');
    process.exit(1);
  }
}

// Run tests
if (import.meta.main) {
  checkServer().then(() => runTests());
}