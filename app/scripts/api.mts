import { createClient } from '@hey-api/openapi-ts';
import { spawnSync } from 'node:child_process';
import { mkdtemp, readdir, readFile, rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { dirname, join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

import config from '../openapi-ts.config.mjs';

const appDir = resolve(dirname(fileURLToPath(import.meta.url)), '..');

async function filesIn(dir: string, prefix = ''): Promise<string[]> {
  const result: string[] = [];
  for (const entry of await readdir(join(dir, prefix), { withFileTypes: true })) {
    const path = join(prefix, entry.name);
    if (entry.isDirectory()) {
      result.push(...await filesIn(dir, path));
    } else {
      result.push(path);
    }
  }
  return result.sort();
}

export async function compareGeneratedDirectories(expectedDir: string, actualDir: string) {
  const expectedFiles = await filesIn(expectedDir);
  const actualFiles = await filesIn(actualDir);
  if (JSON.stringify(expectedFiles) !== JSON.stringify(actualFiles)) {
    throw new Error('Generated client files are stale. Run npm run api:update.');
  }
  for (const file of expectedFiles) {
    const [expected, actual] = await Promise.all([
      readFile(join(expectedDir, file)), readFile(join(actualDir, file)),
    ]);
    if (!expected.equals(actual)) {
      throw new Error(`Generated client is stale: ${file}. Run npm run api:update.`);
    }
  }
}

export async function generateClient(output: string) {
  await createClient({
    ...config,
    input: resolve(appDir, '../backend/api/openapi.json'),
    output,
  });
}

async function main() {
  const mode = process.argv[2];
  if (mode !== 'update' && mode !== 'check' && mode !== 'check-client') {
    throw new Error('Usage: tsx scripts/api.ts update|check|check-client');
  }
  if (mode !== 'check-client') {
    const go = spawnSync(process.env.GO_BINARY || 'go', [
      'run', './cmd/apigen', ...(mode === 'check' ? ['-check'] : []),
    ], { cwd: resolve(appDir, '../backend'), stdio: 'inherit', env: process.env });
    if (go.error) {
      throw new Error(`Cannot run Go. Install Go or use the api-tools Docker service. ${go.error.message}`);
    }
    if (go.status !== 0) {
      throw new Error('Go API generation/check failed. See the output above.');
    }
  }

  const generated = resolve(appDir, 'src/api/generated');
  const temporary = mode === 'update' ? undefined : await mkdtemp(join(tmpdir(), 'catch-me-client-'));
  try {
    await generateClient(temporary || generated);
    if (!temporary) {
      console.log('Updated API document and TypeScript client from Go source.');
      return;
    }
    await compareGeneratedDirectories(temporary, generated);
    console.log('Generated TypeScript client matches the API document.');
  } finally {
    if (temporary) {
      await rm(temporary, { recursive: true, force: true });
    }
  }
}

if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  main().catch((error: unknown) => {
    console.error(error instanceof Error ? error.message : error);
    process.exitCode = 1;
  });
}
