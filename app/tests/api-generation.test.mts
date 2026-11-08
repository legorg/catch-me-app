import assert from 'node:assert/strict';
import { mkdtemp, mkdir, rm, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { dirname, join, resolve } from 'node:path';
import { test } from 'node:test';
import { fileURLToPath } from 'node:url';

import { compareGeneratedDirectories, generateClient } from '../scripts/api.mts';

test('generated client is up to date with the API document', async (t) => {
  const temporary = await mkdtemp(join(tmpdir(), 'catch-me-generation-test-'));
  t.after(() => rm(temporary, { recursive: true, force: true }));
  await generateClient(temporary);
  const appDir = resolve(dirname(fileURLToPath(import.meta.url)), '..');
  await compareGeneratedDirectories(temporary, join(appDir, 'src/api/generated'));
});

test('generation check detects changed, missing, and extra generated files', async (t) => {
  const temporary = await mkdtemp(join(tmpdir(), 'catch-me-stale-test-'));
  t.after(() => rm(temporary, { recursive: true, force: true }));
  const expected = join(temporary, 'expected');
  const actual = join(temporary, 'actual');
  await Promise.all([mkdir(expected), mkdir(actual)]);
  await writeFile(join(expected, 'types.gen.ts'), 'export type Name = string;\n');

  await assert.rejects(compareGeneratedDirectories(expected, actual), /files are stale/);
  await writeFile(join(actual, 'types.gen.ts'), 'export type Name = number;\n');
  await assert.rejects(compareGeneratedDirectories(expected, actual), /client is stale/);
  await writeFile(join(actual, 'types.gen.ts'), 'export type Name = string;\n');
  await compareGeneratedDirectories(expected, actual);
  await writeFile(join(actual, 'extra.gen.ts'), '// outdated file\n');
  await assert.rejects(compareGeneratedDirectories(expected, actual), /files are stale/);
});
