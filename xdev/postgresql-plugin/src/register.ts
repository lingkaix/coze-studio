import { readFileSync } from 'fs';
import { resolve } from 'path';

interface RegisterPluginRequest {
  space_id: string;
  project_id?: string;
  ai_plugin: string;
  openapi: string;
  service_token?: string;
}

interface RegisterPluginResponse {
  code: number;
  msg: string;
  data?: {
    plugin_id: number;
    openapi: string;
  };
}

async function registerPlugin() {
  // Load configuration from environment
  const COZE_API_URL = process.env.COZE_API_URL || 'http://localhost:8080';
  const SPACE_ID = process.env.SPACE_ID;
  const PROJECT_ID = process.env.PROJECT_ID;
  const SERVICE_TOKEN = process.env.API_KEY; // Use the same API key for the service token

  if (!SPACE_ID) {
    console.error('❌ SPACE_ID environment variable is required');
    process.exit(1);
  }

  try {
    // Read the AI plugin manifest and OpenAPI spec
    const aiPluginPath = resolve(__dirname, '../ai_plugin.json');
    const openApiPath = resolve(__dirname, '../openapi.yaml');
    
    const aiPlugin = readFileSync(aiPluginPath, 'utf8');
    const openapi = readFileSync(openApiPath, 'utf8');

    // Prepare the request payload
    const payload: RegisterPluginRequest = {
      space_id: SPACE_ID,
      ai_plugin: aiPlugin,
      openapi: openapi,
    };

    if (PROJECT_ID) {
      payload.project_id = PROJECT_ID;
    }

    if (SERVICE_TOKEN) {
      payload.service_token = SERVICE_TOKEN;
    }

    console.log('🚀 Registering PostgreSQL plugin with Coze Studio...');
    console.log(`📡 API URL: ${COZE_API_URL}`);
    console.log(`🏢 Space ID: ${SPACE_ID}`);
    if (PROJECT_ID) {
      console.log(`📁 Project ID: ${PROJECT_ID}`);
    }

    // Send the registration request
    const response = await fetch(`${COZE_API_URL}/api/developer/register`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(payload),
    });

    const result: RegisterPluginResponse = await response.json();

    if (response.ok && result.code === 0) {
      console.log('✅ Plugin registered successfully!');
      console.log(`🔌 Plugin ID: ${result.data?.plugin_id}`);
      console.log('📋 Next steps:');
      console.log('  1. Start your plugin server with: bun run dev');
      console.log('  2. Update your server URL in the plugin configuration if needed');
      console.log('  3. Test the plugin in Coze Studio');
    } else {
      console.error('❌ Plugin registration failed:');
      console.error(`   Status: ${response.status}`);
      console.error(`   Code: ${result.code}`);
      console.error(`   Message: ${result.msg}`);
      
      if (response.status === 400 && result.msg.includes('invalid')) {
        console.log('\n💡 Common fixes:');
        console.log('   - Check that SPACE_ID is correct');
        console.log('   - Verify that PROJECT_ID exists (if provided)');
        console.log('   - Ensure your OpenAPI spec is valid YAML');
        console.log('   - Check that ai_plugin.json is valid JSON');
      }
      
      process.exit(1);
    }
  } catch (error) {
    console.error('❌ Registration failed with error:', error);
    
    if (error instanceof TypeError && error.message.includes('fetch')) {
      console.log('\n💡 Check that the Coze Studio API is running at:', COZE_API_URL);
    }
    
    process.exit(1);
  }
}

// Run the registration if this script is executed directly
if (import.meta.main) {
  registerPlugin();
}

export { registerPlugin };