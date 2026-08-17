// Fails when a t('...') key used in the source is missing from the locale.
//
// Several of these have shipped as raw keys on screen, because a missing entry
// is invisible until someone opens that particular panel. This turns it into a
// build failure instead of something a user finds.
const fs = require('fs');
const path = require('path');
const os = require('os');

const root = path.join(__dirname, '..', 'src');
const keys = new Set();

(function walk(dir) {
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const full = path.join(dir, entry.name);
    if (entry.isDirectory()) {
      walk(full);
    } else if (/\.(tsx|ts)$/.test(entry.name)) {
      const src = fs.readFileSync(full, 'utf8');
      for (const match of src.matchAll(/\bt\(\s*'([a-zA-Z0-9_.]+)'/g)) {
        keys.add(match[1]);
      }
    }
  }
})(root);

const localePath = path.join(root, 'i18n', 'locales', 'en.ts');
const compiled = fs
  .readFileSync(localePath, 'utf8')
  .replace(/^const en =/, 'module.exports =')
  .replace(/export default en;?\s*$/, '');

const tmp = path.join(os.tmpdir(), `nanokvm-en-${process.pid}.cjs`);
fs.writeFileSync(tmp, compiled);

let translation;
try {
  translation = require(tmp).translation;
} finally {
  fs.unlinkSync(tmp);
}

const has = (key) =>
  key
    .split('.')
    .reduce((node, part) => (node && typeof node === 'object' ? node[part] : undefined), translation) !== undefined;

const missing = [...keys].filter((key) => !has(key)).sort();

if (missing.length > 0) {
  console.error(`${missing.length} translation key(s) used in source but missing from en.ts:`);
  for (const key of missing) console.error(`  ${key}`);
  process.exit(1);
}

console.log(`i18n: ${keys.size} keys checked, none missing`);
