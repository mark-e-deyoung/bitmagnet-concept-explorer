import {defineConfig} from '@playwright/test';

export default defineConfig({
 testDir:'./tests/ui',
 testMatch:'**/crossbrowser-matrix.spec.mjs',
 timeout:30000,
 expect:{timeout:10000},
 workers:2,
 retries:0,
 reporter:[['list'],['html',{outputFolder:'build/playwright-crossbrowser-report',open:'never'}]],
 outputDir:'build/playwright-crossbrowser-results',
 webServer:{
  command:'node scripts/ux/fixture-server.mjs',
  url:'http://127.0.0.1:4175/healthz',
  reuseExistingServer:!process.env.CI,
  timeout:12000
 },
 use:{baseURL:'http://127.0.0.1:4175',trace:'retain-on-failure',screenshot:'only-on-failure'},
 projects:[
  {name:'chromium-desktop',use:{browserName:'chromium',viewport:{width:1440,height:900}}},
  {name:'firefox-desktop',use:{browserName:'firefox',viewport:{width:1440,height:900}}},
  {name:'webkit-desktop',use:{browserName:'webkit',viewport:{width:1440,height:900}}},
  {name:'webkit-mobile',use:{browserName:'webkit',viewport:{width:390,height:844},
   isMobile:true,hasTouch:true}}
 ]
});
