import { startPreviewServer } from "./preview-server.mjs";
import { spawn } from "node:child_process";
import { writeFile } from "node:fs/promises";
import assert from "node:assert/strict";
const directory =
  process.env.FINANCE_PREVIEW_DIR || "/tmp/agrilinks-finance-preview";
const { server, baseURL } = await startPreviewServer(directory, {
  "/private/verification/101": "private-photo.jpg",
  "/private/verification/102": "private-photo.jpg",
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
    "--user-data-dir=/tmp/agrilinks-finance-chrome-" + Date.now(),
    "about:blank",
  ],
  { stdio: ["ignore", "ignore", "pipe"] },
);
let browser;
const timeout = setTimeout(() => {
  chrome.kill();
  process.exitCode = 1;
}, 120000);
try {
  const endpoint = await new Promise((resolve, reject) => {
    let log = "";
    chrome.stderr.on("data", (chunk) => {
      log += chunk;
      const match = log.match(/DevTools listening on (ws:\/\/[^\s]+)/);
      if (match) resolve(match[1]);
    });
    chrome.on("exit", (code) => reject(new Error("Chrome exited " + code)));
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
      "Timed out: " +
        expression +
        "; " +
        (await evaluate("document.body.innerText.slice(-1800)")),
    );
  };
  await call("Page.addScriptToEvaluateOnNewDocument", {
    source: `
    window.previewRequests=[];
    window.previewGPS=false;
    const camera=navigator.mediaDevices.getUserMedia.bind(navigator.mediaDevices);
    navigator.mediaDevices.getUserMedia=async options=>(window.previewStream=await camera(options));
    navigator.geolocation.getCurrentPosition=(success,failure)=>window.previewGPS?success({coords:{latitude:7.7,longitude:8.5,accuracy:12},timestamp:Date.now()}):failure({code:1});
    window.fetch=async(input,options={})=>{
      const path=String(input), body=options.body;
      window.previewRequests.push({path,body:body instanceof FormData?Object.fromEntries([...body].map(([k,v])=>[k,v instanceof File?{size:v.size,type:v.type}:v])):String(body||'')});
      const json=(data,status=200)=>new Response(JSON.stringify(data),{status,headers:{'Content-Type':'application/json'}});
      if(path==='/wallet/evidence/challenge') return json({token:'a'.repeat(64)});
      if(path==='/wallet/evidence') return window.failUpload?json({error:'Simulated upload failure'},400):json({saved:true});
      if(path==='/admin/security') {
        const action=body.get('action');
        if(window.failMFA) return json({error:'The authenticator code is invalid'},400);
        if(action==='enroll'||action==='replace') {window.previewEnrolling=true;return json({secret:'GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ'});}
        const codes=window.previewEnrolling?Array.from({length:10},(_,i)=>i.toString(16).padStart(20,'0')):[];
        window.previewEnrolling=false; return json({verified:true,recoveryCodes:codes});
      }
      if(path.includes('/wallet/banks')) return json([{name:'Test Bank',code:'001'}]);
      if(path.includes('/admin/revenue')) return json({message:'Finance code sent.'});
      return json({revision:'preview'});
    };
  `,
  });
  for (const state of [
    "funding",
    "withdrawal",
    "enrollment",
    "revenue",
    "escrow",
    "admin-security",
    "withdrawal-review",
  ]) {
    for (const width of [320, 390, 768, 1440]) {
      await call("Emulation.setDeviceMetricsOverride", {
        width,
        height: 960,
        deviceScaleFactor: 1,
        mobile: width < 500,
      });
      await call("Page.navigate", { url: baseURL + "/" + state + ".html" });
      for (let i = 0; i < 60; i++) {
        if (
          await evaluate(
            "document.readyState === 'complete' && !!document.querySelector('#root h1')",
          )
        )
          break;
        await new Promise((resolve) => setTimeout(resolve, 50));
      }
      await new Promise((resolve) => setTimeout(resolve, 200));
      await evaluate(
        `document.querySelectorAll('details').forEach(node=>node.open=true)`,
      );
      const layout = await evaluate(
        `({width:innerWidth,scroll:document.documentElement.scrollWidth,text:document.body.innerText})`,
      );
      if (layout.scroll > width)
        throw new Error(
          state + " overflows at " + width + ": " + layout.scroll,
        );
      if (layout.text.includes("NaN") || layout.text.includes("<no value>"))
        throw new Error("Invalid rendered financial data");
      if (state === "withdrawal-review")
        await until(
          `!document.querySelector('option[value="approve"]').disabled`,
        );
      if (
        state === "funding" &&
        (!layout.text.includes("₦10000.00") ||
          !layout.text.includes("₦10190.00"))
      )
        throw new Error("Funding breakdown missing");
      if (
        state === "withdrawal" &&
        (!layout.text.includes("₦10325.00") ||
          !layout.text.includes("Bank account receives"))
      )
        throw new Error("Payout breakdown missing");
      if (
        state === "enrollment" &&
        !(await evaluate(
          `!!document.querySelector('#wallet-pin-setup input[name="current_password"]')`,
        ))
      )
        throw new Error("PIN enrollment missing");
      if (state === "revenue") {
        await evaluate(`document.querySelector('[data-finance-code]').click()`);
        await new Promise((resolve) => setTimeout(resolve, 100));
        if (
          !(await evaluate(
            `document.querySelector('[data-code-status]').textContent.includes('Finance code sent')`,
          ))
        )
          throw new Error("Finance OTP control failed");
      }
      if (width === 390 || (state === "revenue" && width === 1440)) {
        if (state === "funding" || state === "withdrawal")
          await evaluate(
            `document.querySelector('#wallet-quote').scrollIntoView()`,
          );
        const shot = await call("Page.captureScreenshot", { format: "png" });
        await writeFile(
          directory + "/" + state + "-" + width + ".png",
          Buffer.from(shot.data, "base64"),
        );
      }
      console.log(
        JSON.stringify({ state, width, scroll: layout.scroll, passed: true }),
      );
    }
  }
  const navigate = async (state) => {
    await call("Page.navigate", { url: baseURL + "/" + state + ".html" });
    await until(
      `document.readyState==='complete'&&!!document.querySelector('#root h1')`,
    );
    await new Promise((resolve) => setTimeout(resolve, 200));
  };
  await navigate("withdrawal");
  assert.equal(
    await evaluate(
      `(()=>{const form=document.querySelector('.withdrawal-evidence').closest('form');const event=new Event('submit',{bubbles:true,cancelable:true});form.dispatchEvent(event);return event.defaultPrevented&&form.dataset.submitting!=='true';})()`,
    ),
    true,
    "Missing evidence must not lock or submit the form",
  );
  await evaluate(
    `document.querySelector('.withdrawal-evidence input[type=checkbox]').click();document.querySelector('.withdrawal-evidence button').click()`,
  );
  await until(
    `!document.querySelectorAll('.withdrawal-evidence button')[1].disabled`,
  );
  await evaluate(
    `document.querySelectorAll('.withdrawal-evidence button')[1].click()`,
  );
  await until(
    `document.querySelector('.withdrawal-evidence [role=status]').textContent.includes('Allow location')`,
  );
  await evaluate(
    `window.previewGPS=true;window.failUpload=true;document.querySelectorAll('.withdrawal-evidence button')[1].click()`,
  );
  await until(
    `document.querySelector('.withdrawal-evidence [role=status]').textContent.includes('Simulated upload failure')`,
  );
  assert.equal(
    await evaluate(`document.querySelector('[name=evidence_token]').value`),
    "",
  );
  await evaluate(
    `window.failUpload=false;document.querySelectorAll('.withdrawal-evidence button')[1].click()`,
  );
  await until(
    `document.querySelector('[name=evidence_token]').value.length===64`,
  );
  assert.equal(
    await evaluate(
      `window.previewStream.getTracks().every(track=>track.readyState==='ended')`,
    ),
    true,
    "Camera must stop after upload",
  );
  const upload = await evaluate(
    `window.previewRequests.filter(r=>r.path==='/wallet/evidence').at(-1).body`,
  );
  assert.equal(upload.latitude, "7.7");
  assert.equal(upload.longitude, "8.5");
  assert.equal(upload.selfie.type, "image/jpeg");
  assert.ok(upload.selfie.size > 0);
  await evaluate(
    `document.querySelector('.withdrawal-evidence input[type=checkbox]').click()`,
  );
  await until(`!document.querySelector('[name=evidence_token]').value`);
  assert.equal(
    await evaluate(
      `(()=>{const form=document.querySelector('.withdrawal-evidence').closest('form');const event=new Event('submit',{bubbles:true,cancelable:true});form.dispatchEvent(event);return event.defaultPrevented&&form.dataset.submitting!=='true';})()`,
    ),
    true,
    "Revoked evidence must not lock the wallet form",
  );
  await navigate("admin-security");
  await evaluate(
    `document.querySelector('[name=password]').value='preview password';document.querySelector('form').requestSubmit()`,
  );
  await until(`!!document.querySelector('.setup-secret')`);
  await evaluate(
    `window.failMFA=true;document.querySelector('[name=code]').value='123456';document.querySelector('[name=code]').form.requestSubmit()`,
  );
  await until(
    `document.querySelector('[role=alert]')?.textContent.includes('invalid')`,
  );
  assert.equal(
    await evaluate(`!!document.querySelector('.security-primary')`),
    false,
  );
  await evaluate(
    `window.failMFA=false;document.querySelector('[name=code]').form.requestSubmit()`,
  );
  await until(`document.querySelectorAll('.recovery-codes li').length===10`);
  assert.equal(
    await evaluate(
      `document.querySelector('.security-primary').getAttribute('href')`,
    ),
    "/admin/finance",
  );
  // Start an enabled, unverified session to exercise recovery UI independently.
  await navigate("admin-security-enabled");
  await evaluate(
    `[...document.querySelectorAll('button')].find(b=>b.textContent.includes('Use a recovery code')).click()`,
  );
  await until(`!!document.querySelector('[name=password]')`);
  await evaluate(
    `document.querySelector('[name=password]').value='preview password';document.querySelector('[name=code]').value='a'.repeat(20);document.querySelector('form').requestSubmit()`,
  );
  await until(`!!document.querySelector('.security-primary')`);
  assert.ok(
    (await evaluate(`window.previewRequests.at(-1).body`)).includes(
      "action=recover",
    ),
  );
  await navigate("withdrawal-review");
  await until(`!document.querySelector('option[value=approve]').disabled`);
  await evaluate(
    `const select=document.querySelector('select');select.value='approve';select.dispatchEvent(new Event('change',{bubbles:true}));document.querySelector('textarea').value='Reviewed the synthetic test evidence';`,
  );
  await until(`document.querySelector('[name=evidence_reviewed]').required`);
  assert.equal(
    await evaluate(`document.querySelector('form').checkValidity()`),
    false,
    "Approval must require acknowledgment",
  );
  await evaluate(`document.querySelector('[name=evidence_reviewed]').click()`);
  assert.equal(
    await evaluate(`document.querySelector('form').checkValidity()`),
    true,
  );
  await evaluate(
    `document.querySelector('.verification-photos img').src='/missing-private-photo.jpg'`,
  );
  await until(`document.querySelector('option[value=approve]').disabled`);
  assert.equal(
    await evaluate(`document.querySelector('form button').disabled`),
    true,
    "Failed private image must block approval",
  );
  if (exceptions.length) throw new Error(JSON.stringify(exceptions));
  console.log(
    "PASS: 28 responsive cases; private camera/GPS capture, denial/upload failure, consent revocation, MFA enrollment/recovery, private review gating, and no JavaScript exceptions.",
  );
  await send("Browser.close");
} finally {
  clearTimeout(timeout);
  browser?.close();
  chrome.kill();
  await new Promise((resolve) => server.close(resolve));
}
