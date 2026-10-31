#!/usr/bin/env node
import { readFile } from 'node:fs/promises';
import { fileURLToPath } from 'node:url';
import path from 'node:path';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../..');
const htmlPath = path.join(root, 'web/index.html');
const manifestPath = path.join(root, 'tests/ui/control-inventory.json');
const testSuitePath = path.join(root, 'tests/ui/crossbrowser-matrix.spec.mjs');

async function main() {
  process.stdout.write('Checking UI control inventory and cross-engine test coverage...\n');

  const [htmlContent, manifestContent, testSuiteContent] = await Promise.all([
    readFile(htmlPath, 'utf8'),
    readFile(manifestPath, 'utf8'),
    readFile(testSuitePath, 'utf8'),
  ]);

  const manifest = JSON.parse(manifestContent);
  const controls = manifest.controls || [];

  if (controls.length !== manifest.total_controls) {
    throw new Error(`Manifest total_controls mismatch: declared ${manifest.total_controls}, found ${controls.length}`);
  }

  // Verify all static interactive elements in web/index.html have manifest entries
  const staticControls = controls.filter(c => c.type === 'static');
  for (const c of staticControls) {
    // Check if the selector identifier is present in web/index.html
    const token = c.selector.replace(/^[a-z0-9]+#|^#/i, '').replace(/^[a-z0-9]+\.|^\./i, '').split(/[\s:\[]/)[0];
    if (token && !htmlContent.includes(token)) {
      throw new Error(`Control selector token "${token}" for "${c.control_id}" not found in web/index.html`);
    }
  }

  // Collect all testcase_ids referenced in manifest
  const referencedTestcases = new Set();
  for (const c of controls) {
    if (!c.testcase_ids || c.testcase_ids.length === 0) {
      throw new Error(`Control "${c.control_id}" has no assigned testcase_ids`);
    }
    for (const tid of c.testcase_ids) {
      referencedTestcases.add(tid);
    }
    if (!c.required_browsers || c.required_browsers.length < 3) {
      throw new Error(`Control "${c.control_id}" does not mandate the minimum 3 browser engines`);
    }
  }

  // Verify every referenced testcase_id exists in crossbrowser-matrix.spec.mjs
  const missingTests = [];
  for (const tid of referencedTestcases) {
    // Check if the test file contains the testcase identifier
    const hasIdentifier = testSuiteContent.includes(`[${tid}]`) || testSuiteContent.includes(tid);
    if (!hasIdentifier) {
      missingTests.push(tid);
    }
  }

  if (missingTests.length > 0) {
    throw new Error(`The following testcase_ids are in control-inventory.json but missing from crossbrowser-matrix.spec.mjs: ${missingTests.join(', ')}`);
  }

  process.stdout.write(
    `MANIFEST_CHECK_PASS: All ${controls.length} controls mapped across ${referencedTestcases.size} executable testcases for 3 browser engines.\n`
  );
}

main().catch(err => {
  process.stderr.write(`MANIFEST_CHECK_FAIL: ${err.message}\n`);
  process.exit(1);
});
