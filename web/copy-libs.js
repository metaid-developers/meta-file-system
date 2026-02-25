/**
 * Copy min.js files from node_modules to web/ for browser loading.
 * Run: npm run copy-libs (or automatically after npm install via postinstall)
 */
const fs = require('fs');
const path = require('path');

const root = __dirname;
const nodeModules = path.join(root, 'node_modules');

const copies = [
  ['meta-contract/dist/metaContract.browser.min.js', 'metacontract.min.js'],
  ['@metaid/metaid/dist/metaid.iife.js', 'metaid.min.js'],
  ['bitcoinjs-lib-browser/bitcoinjs.min.js', 'bitcoinjs.min.js'],
];

for (const [from, to] of copies) {
  const srcPath = path.join(nodeModules, from);
  const destPath = path.join(root, to);
  if (fs.existsSync(srcPath)) {
    fs.copyFileSync(srcPath, destPath);
    console.log('✓ Copied', from, '->', to);
  } else {
    console.warn('⚠ Skip (not found):', from);
  }
}
