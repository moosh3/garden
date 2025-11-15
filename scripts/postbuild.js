const fs = require('fs');
const path = require('path');

const projectRoot = path.join(__dirname, '..');
const source = path.join(projectRoot, '.next', 'routes-manifest.json');
const exportDir = path.join(projectRoot, 'out');
const target = path.join(exportDir, 'routes-manifest.json');

if (!fs.existsSync(source)) {
  console.log('[postbuild] No routes-manifest found in .next, skipping copy.');
  process.exit(0);
}

if (!fs.existsSync(exportDir)) {
  console.log('[postbuild] No out directory found, skipping copy.');
  process.exit(0);
}

fs.copyFileSync(source, target);
console.log('[postbuild] Copied routes-manifest.json to out/.');

