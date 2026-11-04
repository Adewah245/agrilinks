import assert from "node:assert/strict";
import { spawn } from "node:child_process";
import { readdir, writeFile } from "node:fs/promises";
import { resolve } from "node:path";
import { startPreviewServer } from "./preview-server.mjs";

const directory =
  process.env.REACT_PREVIEW_DIR || "/tmp/agrilinks-react-preview";
const cases = (await readdir(directory))
  .filter(
    (name) => name.startsWith("TestReactPages-") && name.endsWith(".html"),
  )
  .sort();
assert.equal(
  cases.length,
  33,
  "Generate all page fixtures with REACT_PREVIEW_DIR set when running the Go handler tests.",
);
const { server, baseURL } = await startPreviewServer(directory, {
  "/login": "TestReactPages-login-html.html",
});
const chrome = spawn(
  process.env.CHROME || "/usr/bin/google-chrome",
  [
    "--headless",
    "--no-sandbox",
    "--disable-gpu",
    "--disable-background-networking",
    "--no-first-run",
    "--use-fake-device-for-media-stream",
    "--use-fake-ui-for-media-stream",
    "--remote-debugging-port=0",
    `--user-data-dir=/tmp/agrilinks-react-chrome-${Date.now()}`,
    "about:blank",
  ],
  { stdio: ["ignore", "ignore", "pipe"] },
);
let browser;
const timeout = setTimeout(() => {
  chrome.kill();
  server.close();
  process.exitCode = 1;
}, 180000);
try {
  const endpoint = await new Promise((resolve, reject) => {
    let log = "";
    chrome.stderr.on("data", (chunk) => {
      log += chunk;
      const match = log.match(/DevTools listening on (ws:\/\/[^\s]+)/);
      if (match) resolve(match[1]);
    });
    chrome.on("exit", (code) =>
      reject(new Error(`Chrome exited ${code}: ${log}`)),
    );
  });
  browser = new WebSocket(endpoint);
  await new Promise((resolve) =>
    browser.addEventListener("open", resolve, { once: true }),
  );
  let sequence = 0;
  const pending = new Map();
  const exceptions = [];
  browser.addEventListener("message", (event) => {
    const response = JSON.parse(event.data);
    if (response.id) {
      const entry = pending.get(response.id);
      pending.delete(response.id);
      if (response.error)
        entry.reject(new Error(JSON.stringify(response.error)));
      else entry.resolve(response.result);
    } else if (response.method === "Runtime.exceptionThrown")
      exceptions.push(response.params.exceptionDetails);
  });
  const send = (method, params = {}, sessionId) =>
    new Promise((resolve, reject) => {
      const id = ++sequence;
      pending.set(id, { resolve, reject });
      browser.send(JSON.stringify({ id, method, params, sessionId }));
    });
  const { targetId } = await send("Target.createTarget", {
    url: "about:blank",
  });
  const { sessionId } = await send("Target.attachToTarget", {
    targetId,
    flatten: true,
  });
  const call = (method, params = {}) => send(method, params, sessionId);
  await call("Page.enable");
  await call("Runtime.enable");
  const evaluate = async (expression) => {
    const response = await call("Runtime.evaluate", {
      expression,
      returnByValue: true,
      awaitPromise: true,
    });
    if (response.exceptionDetails)
      throw new Error(JSON.stringify(response.exceptionDetails));
    return response.result.value;
  };
  const until = async (expression) => {
    for (let i = 0; i < 160; i++) {
      if (await evaluate(expression)) return;
      await new Promise((resolve) => setTimeout(resolve, 50));
    }
    throw new Error(
      `Timed out: ${expression}; ${JSON.stringify(await evaluate("({url:location.href,nav:document.querySelector('.navbar')?.outerHTML,errors:document.body.innerText.slice(0,120)})"))}`,
    );
  };
  await call("Page.addScriptToEvaluateOnNewDocument", {
    source: `
    window.testRequests = [];
    window.fetch = async (input, options = {}) => {
      const path = new URL(String(input), location.origin).pathname;
      window.testRequests.push({path, body: options.body instanceof FormData ? Object.fromEntries([...options.body].map(([key, value]) => [key, value instanceof File ? {size:value.size,type:value.type} : value])) : String(options.body || '')});
      const json = value => new Response(JSON.stringify(value), {headers:{'Content-Type':'application/json'}});
      if (path === '/api/states') return json([{id:1,name:'Benue'}]);
      if (path.endsWith('/lgas')) return json([{id:1,name:'Makurdi'}]);
      if (path === '/wallet/banks') return json([{name:'Example Bank',code:'001'}]);
      if (path === '/wallet/status') return json({revision:JSON.parse(document.getElementById('page-data').textContent).data.Revision});
      if (path === '/chat/presence') return json({online_user_ids:[1]});
      if (path === '/learning/chat') return json({answer:'Start with healthy soil and a small planting area.'});
      if (path.startsWith('/register/phone-')) return json({message:'Phone confirmation ready.',retry_after:1});
      if (path === '/register') { const response = json({}); Object.defineProperties(response,{redirected:{value:true},url:{value:location.origin+'/login'}}); return response; }
      if (path === '/admin/revenue') return json({message:'Finance code sent.'});
      return json({});
    };
    navigator.geolocation.getCurrentPosition = (success, failure) => failure({code:1,message:'Permission declined for preview'});
    window.confirm = () => true;
  `,
  });
  const navigate = async (name) => {
    await call("Page.navigate", { url: baseURL + "/" + name });
    await until(
      `document.readyState === 'complete' && !!document.querySelector('#root h1, #root h2')`,
    );
    await new Promise((resolve) => setTimeout(resolve, 100));
    assert.equal(
      await evaluate(
        `document.body.innerText.includes('This page could not be loaded.')`,
      ),
      false,
      name,
    );
    assert.equal(
      await evaluate(
        `document.body.innerText.includes('[object Object]') || document.body.innerText.includes('NaN') || document.body.innerText.includes('<no value>')`,
      ),
      false,
      name,
    );
  };
  const overflows = [];
  for (const width of [390, 1440]) {
    await call("Emulation.setDeviceMetricsOverride", {
      width,
      height: 960,
      deviceScaleFactor: 1,
      mobile: width < 500,
    });
    for (const name of cases) {
      await navigate(name);
      const scroll = await evaluate("document.documentElement.scrollWidth");
      if (scroll > width + 1) overflows.push({ name, width, scroll });
      if (
        width === 390 &&
        (await evaluate(`!!document.querySelector('.nav-toggle')`))
      ) {
        await evaluate(`document.querySelector('.nav-toggle').click()`);
        await until(
          `document.querySelector('.nav-toggle').getAttribute('aria-expanded') === 'true'`,
        );
        await evaluate(
          `document.dispatchEvent(new KeyboardEvent('keydown',{key:'Escape',bubbles:true}))`,
        );
        await until(
          `document.querySelector('.nav-toggle').getAttribute('aria-expanded') === 'false'`,
        );
      }
      if (width === 390 && /dashboard|wallet-html|register/.test(name)) {
        const shot = await call("Page.captureScreenshot", { format: "png" });
        await writeFile(
          resolve(directory, name.replace(".html", `-${width}.png`)),
          Buffer.from(shot.data, "base64"),
        );
      }
    }
    console.log(
      `Rendered all 33 React pages at ${width}px; mobile navigation checked.`,
    );
  }
  await navigate("TestReactPages-login-html.html");
  await evaluate(`document.querySelector('#login-form').requestSubmit()`);
  await until(
    `document.querySelector('.message')?.textContent.includes('Phone number is required')`,
  );
  await evaluate(`document.querySelector('#toggle-password').click()`);
  await until(`document.querySelector('#password').type === 'text'`);
  await navigate("TestReactPages-forgot-password-html.html");
  await evaluate(`document.querySelector('.toggle-password').click()`);
  await until(`document.querySelector('#new_password').type === 'text'`);
  await navigate("TestReactPages-register-html.html");
  await until(`document.querySelector('#state option[value="Benue"]')`);
  await evaluate(
    `document.querySelector('input[name="purpose"][value="buy"]').click(); for(const [id,value] of Object.entries({'first-name':'Ada','last-name':'Farmer',email:'ada@example.test',phone:'08012345678',community:'Makurdi',password:'sample password','confirm-password':'sample password'})) { const input=document.getElementById(id);input.value=value;input.dispatchEvent(new Event('input',{bubbles:true})); } const state=document.getElementById('state');state.value='Benue';state.dispatchEvent(new Event('change',{bubbles:true}));`,
  );
  await until(`document.querySelector('#lga option[value="Makurdi"]')`);
  await evaluate(
    `document.getElementById('lga').value='Makurdi';document.getElementById('register-form').requestSubmit()`,
  );
  await until(`!document.querySelector('.profile-section').hidden`);
  await evaluate(`document.getElementById('open-selfie-camera').click()`);
  await until(`!document.getElementById('take-selfie').disabled`);
  await evaluate(`document.getElementById('take-selfie').click()`);
  await until(`!document.getElementById('selfie-preview').hidden`);
  await evaluate(`document.getElementById('send-registration-code').click()`);
  await until(`!document.getElementById('registration-code').disabled`);
  await evaluate(
    `const input=document.getElementById('registration-code');input.value='123456';input.dispatchEvent(new Event('input',{bubbles:true}));document.getElementById('verify-registration-phone').click()`,
  );
  await until(`!document.getElementById('registration-pin-section').hidden`);
  await evaluate(
    `for(const id of ['transaction-pin','transaction-pin-confirmation']) {const input=document.getElementById(id);input.value='582491';input.dispatchEvent(new Event('input',{bubbles:true}));}`,
  );
  assert.equal(
    await evaluate(`document.getElementById('complete-registration').disabled`),
    false,
  );
  await evaluate(`document.getElementById('register-form').requestSubmit()`);
  await until(`location.pathname === '/login'`);
  console.log(
    "Login validation, password visibility, selfie capture, phone confirmation, PIN gating, multipart registration and redirect passed.",
  );
  await navigate('TestReactPages-storage-html.html');
  await evaluate(`document.getElementById('add-btn').click()`);
  await until(`document.getElementById('modal').classList.contains('modal-open') && document.getElementById('form-action').value === 'create'`);
  assert.equal(await evaluate(`document.getElementById('product-id').value`), '');
  assert.equal(await evaluate(`document.getElementById('produce').value`), '');
  await evaluate(`document.getElementById('capture-location').click()`);
  await until(`document.getElementById('location-status').textContent.includes('Allow location access')`);
  await evaluate(`document.getElementById('close-btn').click()`);
  await until(`!document.getElementById('modal').classList.contains('modal-open')`);
  await navigate('TestReactPages-marketplace-html.html');
  await evaluate(`const input=document.getElementById('search');Object.getOwnPropertyDescriptor(HTMLInputElement.prototype,'value').set.call(input,'no matching product');input.dispatchEvent(new Event('input',{bubbles:true}));`);
  await until(`document.querySelectorAll('#product-grid .product-card').length === 0`);
  await navigate('TestReactPages-profile-html.html');
  await evaluate(`document.getElementById('edit-profile-btn').click()`);
  await until(`getComputedStyle(document.getElementById('edit-modal')).display === 'flex'`);
  await evaluate(`document.getElementById('cancel-edit-btn').click()`);
  await until(`getComputedStyle(document.getElementById('edit-modal')).display === 'none'`);
  await navigate('TestReactPages-wallet-html.html');
  await evaluate(`document.getElementById('wallet-balance-toggle').click()`);
  await until(`document.getElementById('wallet-balance-value').textContent === '••••••'`);
  await evaluate(`document.querySelector('[data-wallet-finish="silver"]').click()`);
  await until(`document.getElementById('wallet-metal-card').dataset.finish === 'silver'`);
  console.log('Storage editor and GPS denial, marketplace search, profile editing, and wallet controls passed.');
  assert.deepEqual(exceptions, [], "Browser JavaScript exceptions");
  assert.deepEqual(overflows, [], "Pages overflow their viewport");
  console.log("PASS: 66 page layouts and interactive authentication flows.");
  await send("Browser.close");
} finally {
  clearTimeout(timeout);
  browser?.close();
  chrome.kill();
  await new Promise((resolve) => server.close(resolve));
}
