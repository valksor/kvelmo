/**
 * Test server helper for starting and stopping the Go test server.
 *
 * This module spawns the Go binary as a child process and manages its lifecycle.
 * The server is started on a random available port for test isolation.
 */

import { spawn, ChildProcess } from 'child_process';
import { dirname, join } from 'path';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

export interface TestServerOptions {
  /**
   * Path to the Go server binary. Defaults to MEHR_SERVER_PATH env var
   * or a built binary in the project.
   */
  serverPath?: string;

  /**
   * Working directory for the server. Defaults to a temp directory.
   */
  workDir?: string;

  /**
   * Port to run on. Defaults to 0 (random available port).
   */
  port?: number;

  /**
   * Environment variables to pass to the server.
   */
  env?: Record<string, string>;
}

export interface TestServer {
  /**
   * The child process for the server.
   */
  process: ChildProcess;

  /**
   * The URL of the running server (includes port).
   */
  url: string;

  /**
   * The port the server is listening on.
   */
  port: number;

  /**
   * Clean shutdown of the server.
   */
  stop: () => Promise<void>;
}

let serverInstance: TestServer | null = null;

/**
 * Start the Go test server.
 *
 * The server will be started in project mode with a temporary workspace.
 * It uses the `--port=0` flag to get a random available port.
 *
 * @example
 * ```ts
 * import { startTestServer } from './helpers/server';
 *
 * const server = await startTestServer();
 * console.log(`Server running at ${server.url}`);
 *
 * // After tests...
 * await server.stop();
 * ```
 */
export async function startTestServer(options: TestServerOptions = {}): Promise<TestServer> {
  // Return existing instance if already running
  if (serverInstance) {
    return serverInstance;
  }

  const { serverPath = process.env.MEHR_SERVER_PATH, workDir, port = 0, env = {} } = options;

  // Determine server binary path - use project build output
  const binaryPath = serverPath || join(__dirname, '../../build/mehr');

  // Prepare environment variables
  const serverEnv = {
    ...process.env,
    MEHR_TEST_MODE: '1',
    MEHR_DISABLE_AUTH: '1',
    ...env,
  };

  // Build server command
  const args = ['serve', '--port', port.toString()];

  // Spawn the server process
  const serverProcess = spawn(binaryPath, args, {
    env: serverEnv,
    stdio: ['ignore', 'pipe', 'pipe'],
    cwd: workDir,
  });

  // Create a promise that resolves when the server starts
  const serverReady = new Promise<TestServer>((resolve, reject) => {
    let output = '';
    let portFound = false;

    const timeout = setTimeout(() => {
      serverProcess.kill();
      reject(new Error(`Server startup timeout. Output:\n${output}`));
    }, 30000); // 30 second timeout

    serverProcess.stdout?.on('data', (data: Buffer) => {
      const text = data.toString();
      output += text;

      // Look for the "server started" message with port
      // Output format: "INFO server started port=59002 mode=project"
      // Match against accumulated output in case data comes in chunks
      const portMatch = output.match(/port[= ](\d+)/);
      if (portMatch) {
        const serverPort = parseInt(portMatch[1], 10);
        portFound = true;

        clearTimeout(timeout);
        const url = `http://localhost:${serverPort}`;

        const testServer: TestServer = {
          process: serverProcess,
          url,
          port: serverPort,
          stop: async () => {
            if (serverInstance === null) {
              return;
            }
            serverInstance = null;
            serverProcess.kill('SIGTERM');

            // Wait for process to exit
            await new Promise<void>((resolve) => {
              const cleanup = setTimeout(() => {
                serverProcess.kill('SIGKILL');
                resolve();
              }, 5000);

              serverProcess.on('exit', () => {
                clearTimeout(cleanup);
                resolve();
              });
            });
          },
        };

        serverInstance = testServer;
        resolve(testServer);
      }
    });

    serverProcess.stderr?.on('data', (data: Buffer) => {
      output += data.toString();
    });

    serverProcess.on('error', (err) => {
      clearTimeout(timeout);
      reject(new Error(`Failed to start server: ${err.message}`));
    });

    serverProcess.on('exit', (code, signal) => {
      if (!portFound) {
        clearTimeout(timeout);
        reject(new Error(`Server exited with code ${code} (${signal})\nOutput:\n${output}`));
      }
    });
  });

  return serverReady;
}

/**
 * Get or create the global test server instance.
 *
 * This is useful for Playwright's `globalSetup` to ensure a single server
 * instance is shared across all tests.
 */
export async function getGlobalServer(options?: TestServerOptions): Promise<TestServer> {
  if (serverInstance) {
    return serverInstance;
  }

  // Try to get port and URL from environment if set
  if (process.env.TEST_SERVER_URL) {
    const url = new URL(process.env.TEST_SERVER_URL);
    const port = parseInt(url.port, 10);

    serverInstance = {
      process: null as unknown as ChildProcess, // No process to manage
      url: process.env.TEST_SERVER_URL,
      port,
      stop: () => {
        serverInstance = null;
        return Promise.resolve();
      },
    };

    return serverInstance;
  }

  return startTestServer(options);
}

/**
 * Stop the global test server if running.
 */
export async function stopGlobalServer(): Promise<void> {
  if (serverInstance) {
    await serverInstance.stop();
  }
}

/**
 * Read the server port from a running server's output.
 *
 * This is an alternative method if the server was started externally.
 */
export async function discoverServerPort(): Promise<number | null> {
  // For external server discovery, could read from a known file
  // or try to connect to default ports
  const defaultPorts = [8080, 8081, 3000, 3001];

  for (const port of defaultPorts) {
    try {
      const response = await fetch(`http://localhost:${port}/api/v1/health`, {
        signal: AbortSignal.timeout(500),
      });
      if (response.ok) {
        return port;
      }
    } catch {
      // Port not available, try next
    }
  }

  return null;
}
