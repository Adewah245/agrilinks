import { startPreviewServer } from './preview-server.mjs';
import { spawn } from 'node:child_process';
import { writeFile } from 'node:fs/promises';
const directory = process.env.FINANCE_PREVIEW_DIR || '/tmp/agrilinks-finance-preview';
const { server, baseURL } = await startPreviewServer(directory);
const chrome = spawn(process.env.CHROME || '/usr/bin/google-chrome', ['--headless', '--no-sandbox', '--disable-gpu', '--disable-background-networking', '--no-first-run', '--allow-file-access-from-files', '--remote-debugging-port=0', '--user-data-dir=/tmp/agrilinks-finance-chrome-' + Date.now(), 'about:blank'], { stdio: ['ignore', 'ignore', 'pipe'] });
let browser;
const timeout = setTimeout(() => { chrome.kill(); process.exitCode = 1; }, 60000);
try {
  const endpoint = await new Promise((resolve, reject) => {
    let log = '';
    chrome.stderr.on('data', chunk => { log += chunk; const match = log.match(/DevTools listening on (ws:\/\/[^\s]+)/); if (match) resolve(match[1]); });
    chrome.on('exit', code => reject(new Error('Chrome exited ' + code)));
  });
  browser = new WebSocket(endpoint);
  await new Promise(resolve => browser.addEventListener('open', resolve, { once: true }));
  let sequence = 0; const pending = new Map(); const exceptions = [];
  browser.addEventListener('message', event => {
    const response = JSON.parse(event.data);
    if (response.id) { const entry = pending.get(response.id); pending.delete(response.id); if (response.error) entry.reject(new Error(JSON.stringify(response.error))); else entry.resolve(response.result); }
    else if (response.method === 'Runtime.exceptionThrown') exceptions.push(response.params.exceptionDetails);
  });
  const send = (method, params = {}, sessionId) => new Promise((resolve, reject) => { const id = ++sequence; pending.set(id, { resolve, reject }); browser.send(JSON.stringify({ id, method, params, sessionId })); });
  const { targetId } = await send('Target.createTarget', { url: 'about:blank' });
  const { sessionId } = await send('Target.attachToTarget', { targetId, flatten: true });
  const call = (method, params = {}) => send(method, params, sessionId);
  await call('Page.enable'); await call('Runtime.enable');
  const evaluate = async expression => {
    const response = await call('Runtime.evaluate', { expression, returnByValue: true, awaitPromise: true });
    if (response.exceptionDetails) throw new Error(JSON.stringify(response.exceptionDetails));
    return response.result.value;
  };
  await call('Page.addScriptToEvaluateOnNewDocument', { source: `window.fetch=async(input)=>new Response(JSON.stringify(String(input).includes('/wallet/banks')?[{name:'Test Bank',code:'001'}]:String(input).includes('/admin/revenue')?{message:'Finance code sent.'}:{revision:'preview'}),{headers:{'Content-Type':'application/json'}});` });
  for (const state of ['funding', 'withdrawal', 'enrollment', 'revenue', 'escrow']) {
    for (const width of [320, 390, 768, 1440]) {
      await call('Emulation.setDeviceMetricsOverride', { width, height: 960, deviceScaleFactor: 1, mobile: width < 500 });
      await call('Page.navigate', { url: baseURL + '/' + state + '.html' });
      for (let i = 0; i < 60; i++) { if (await evaluate("document.readyState === 'complete' && !!document.querySelector('#root h1')")) break; await new Promise(resolve => setTimeout(resolve, 50)); }
      await new Promise(resolve => setTimeout(resolve, 200));
      await evaluate(`document.querySelectorAll('details').forEach(node=>node.open=true)`);
      const layout = await evaluate(`({width:innerWidth,scroll:document.documentElement.scrollWidth,text:document.body.innerText})`);
      if (layout.scroll > width) throw new Error(state + ' overflows at ' + width + ': ' + layout.scroll);
      if (layout.text.includes('NaN') || layout.text.includes('<no value>')) throw new Error('Invalid rendered financial data');
      if (state === 'funding' && (!layout.text.includes('₦10000.00') || !layout.text.includes('₦10190.00'))) throw new Error('Funding breakdown missing');
      if (state === 'withdrawal' && (!layout.text.includes('₦10325.00') || !layout.text.includes('Bank account receives'))) throw new Error('Payout breakdown missing');
      if (state === 'enrollment' && !await evaluate(`!!document.querySelector('#wallet-pin-setup input[name="current_password"]')`)) throw new Error('PIN enrollment missing');
      if (state === 'revenue') {
        await evaluate(`document.querySelector('[data-finance-code]').click()`);
        await new Promise(resolve => setTimeout(resolve, 100));
        if (!await evaluate(`document.querySelector('[data-code-status]').textContent.includes('Finance code sent')`)) throw new Error('Finance OTP control failed');
      }
      if (width === 390 || (state === 'revenue' && width === 1440)) {
        if (state === 'funding' || state === 'withdrawal') await evaluate(`document.querySelector('#wallet-quote').scrollIntoView()`);
        const shot = await call('Page.captureScreenshot', { format: 'png' });
        await writeFile(directory + '/' + state + '-' + width + '.png', Buffer.from(shot.data, 'base64'));
      }
      console.log(JSON.stringify({ state, width, scroll: layout.scroll, passed: true }));
    }
  }
  if (exceptions.length) throw new Error(JSON.stringify(exceptions));
  console.log('PASS: 20 responsive cases; fee breakdowns, PIN enrollment, treasury OTP control and no JavaScript exceptions.');
  await send('Browser.close');
} finally { clearTimeout(timeout); browser?.close(); chrome.kill(); await new Promise(resolve => server.close(resolve)); }
