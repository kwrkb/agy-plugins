#!/usr/bin/env node
import { spawnSync, execFileSync } from 'node:child_process';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = dirname(fileURLToPath(import.meta.url));

let token = process.env.GITHUB_PERSONAL_ACCESS_TOKEN
  || process.env.GITHUB_TOKEN
  || process.env.GH_TOKEN;

if (!token) {
  try {
    token = execFileSync('gh', ['auth', 'token'], { encoding: 'utf8' }).trim();
  } catch {
    process.stderr.write(
      'github-mcp-wrapper: failed to resolve token.\n' +
      'Set GITHUB_PERSONAL_ACCESS_TOKEN or run: gh auth login\n'
    );
    process.exit(1);
  }
}

const ext = process.platform === 'win32' ? '.exe' : '';
const bin = join(__dirname, `github-mcp-server${ext}`);

const result = spawnSync(bin, ['stdio', ...process.argv.slice(2)], {
  stdio: 'inherit',
  env: { ...process.env, GITHUB_PERSONAL_ACCESS_TOKEN: token },
});

process.exit(result.status ?? 1);
