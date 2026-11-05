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
  35,
  "Generate all page fixtures with REACT_PREVIEW_DIR set when running the Go handler tests.",
);
const { server, baseURL } = await startPreviewServer(directory, {
  "/login": "TestReactPages-login-html.html",
  "/learning": "TestPublicLearningRoutesNeedNoAccountOrDatabase.html",
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
}, 420000);
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
      if (!entry) return;
      pending.delete(response.id);
      clearTimeout(entry.timer);
      if (response.error)
        entry.reject(new Error(JSON.stringify(response.error)));
      else entry.resolve(response.result);
    } else if (response.method === "Runtime.exceptionThrown")
      exceptions.push(response.params.exceptionDetails);
  });
  const send = (method, params = {}, sessionId) =>
    new Promise((resolve, reject) => {
      const id = ++sequence;
      const timer = setTimeout(() => {
        pending.delete(id);
        reject(new Error(`Browser command timed out: ${method}`));
      }, 20000);
      pending.set(id, { resolve, reject, timer });
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
  await call("Network.enable");
  await call("Network.setBlockedURLs", {
    urls: [
      "https://cdnjs.cloudflare.com/*",
      "https://fonts.googleapis.com/*",
      "https://fonts.gstatic.com/*",
    ],
  });
  await call("Emulation.setEmulatedMedia", {
    features: [{ name: "prefers-reduced-motion", value: "reduce" }],
  });
  const evaluate = async (expression) => {
    const response = await call("Runtime.evaluate", {
      expression: `{ ${expression}\n }`,
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
    window.spokenAnswers = [];
    class TestRecognition {
      start() { window.testRecognition = this; }
      stop() { this.onend?.(); }
      abort() { this.onend?.(); }
    }
    window.SpeechRecognition = location.search.includes('recorder=1') ? undefined : TestRecognition;
    window.webkitSpeechRecognition = undefined;
    window.SpeechSynthesisUtterance = class { constructor(text) { this.text = text; } };
    Object.defineProperty(window, 'speechSynthesis', {value:{
      getVoices: () => [{lang:'en-NG'}],
      cancel: () => { window.speechCancelled = true; },
      speak: utterance => { window.spokenAnswers.push(utterance.text); window.lastUtterance = utterance; },
    }, configurable:true});
    window.fetch = async (input, options = {}) => {
      const path = new URL(String(input), location.origin).pathname;
      window.testRequests.push({path, body: options.body instanceof FormData ? Object.fromEntries([...options.body].map(([key, value]) => [key, value instanceof File ? {size:value.size,type:value.type} : value])) : String(options.body || '')});
      const json = value => new Response(JSON.stringify(value), {headers:{'Content-Type':'application/json'}});
      if (path === '/api/states') return json([{id:1,name:'Benue'}]);
      if (path.endsWith('/lgas')) return json([{id:1,name:'Makurdi'}]);
      if (path === '/wallet/banks') return json([{name:'Example Bank',code:'001'}]);
      if (path === '/wallet/status') return json({revision:JSON.parse(document.getElementById('page-data').textContent).data.Revision});
      if (path === '/chat/presence') return json({online_user_ids:[1]});
      if (path === '/learning/chat') {
        if (window.holdLearning) await new Promise(resolve => { window.releaseLearning = resolve; });
        if (window.failLearning) return new Response(JSON.stringify({error:'Please try again.'}), {status:503,headers:{'Content-Type':'application/json'}});
        return json({answer:'**Healthy soil** helps your plants grow. <img src=x onerror=alert(1)>'});
      }
      if (path === '/learning/transcribe') {
        if (window.holdTranscription) await new Promise(resolve => { window.releaseTranscription = resolve; });
        return json({text:'How do I plant maize?'});
      }
      if (path === '/learning/example' || path === '/ai-assistant/example') {
        if (window.failImages) return new Response(JSON.stringify({error:'The example picture is unavailable. Please try again.'}), {status:503,headers:{'Content-Type':'application/json'}});
        return json({data_url:'data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+jV8sAAAAASUVORK5CYII=',alt:'Example farming picture',caption:'Example drawing. It cannot confirm what is wrong on your farm.'});
      }
      if (path === '/ai-assistant') {
        if (window.holdDiagnosis) await new Promise(resolve => { window.releaseDiagnosis = resolve; });
        if (window.failDiagnosis) return new Response(JSON.stringify({error:'Please try again.'}), {status:503,headers:{'Content-Type':'application/json'}});
        return json({result:'Check the soil. Add water only if it feels dry. <img src=x onerror=alert(1)>',diagnosis_id:42,examples_available:true});
      }
      if (path.startsWith('/register/phone-')) return json({message:'Phone confirmation ready.',retry_after:1});
      if (path === '/register') { sessionStorage.setItem('registrationRequest', JSON.stringify(window.testRequests.at(-1).body)); const response = json({}); Object.defineProperties(response,{redirected:{value:true},url:{value:location.origin+'/login'}}); return response; }
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
      if (
        /dashboard|wallet-html|register|admin-html|ai-assistant-html|deliveries|delivery_partner|wallet_admin|platform_revenue|market-insights|learn-html|profile-html/.test(
          name,
        )
      ) {
        const shot = await call("Page.captureScreenshot", { format: "png" });
        await writeFile(
          resolve(directory, name.replace(".html", `-${width}.png`)),
          Buffer.from(shot.data, "base64"),
        );
      }
    }
    console.log(
      `Rendered all 35 React pages at ${width}px; mobile navigation checked.`,
    );
  }
  for (const width of [390, 1440]) {
    await call("Emulation.setDeviceMetricsOverride", {
      width,
      height: 960,
      deviceScaleFactor: 1,
      mobile: width < 500,
    });
    await navigate("TestPublicLearningRoutesNeedNoAccountOrDatabase.html");
    assert.equal(
      await evaluate(`document.documentElement.scrollWidth <= ${width + 1}`),
      true,
      "Public learning must fit the screen",
    );
    assert.equal(
      await evaluate(
        `document.body.innerText.includes('No account is needed')`,
      ),
      true,
    );
    await evaluate(`document.getElementById('openLearnHub').click()`);
    await until(`document.getElementById('learnModal').open`);
    assert.equal(
      await evaluate(`document.querySelectorAll('.learn-topic-card').length`),
      8,
    );
    assert.equal(
      await evaluate(`document.activeElement.id`),
      "learnSearchInput",
    );
    assert.equal(
      await evaluate(
        `document.getElementById('learnModal').scrollWidth <= document.getElementById('learnModal').clientWidth + 1`,
      ),
      true,
    );
    await evaluate(
      `const input=document.getElementById('learnSearchInput'); Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, 'value').set.call(input,'goat'); input.dispatchEvent(new Event('input',{bubbles:true}));`,
    );
    await until(`document.querySelectorAll('.learn-topic-card').length === 1`);
    await evaluate(`document.querySelector('.learn-topic-card').click()`);
    await until(
      `document.getElementById('learnModalTitle').textContent === 'Goat farming'`,
    );
    assert.equal(
      await evaluate(
        `document.querySelectorAll('.learn-lesson-section').length`,
      ),
      4,
    );
    assert.equal(
      await evaluate(
        `new URL(document.querySelector('#learnLessonView a').href).searchParams.get('topic')`,
      ),
      "Goat farming",
    );
    const shot = await call("Page.captureScreenshot", { format: "png" });
    await writeFile(
      resolve(directory, `learning-guide-${width}.png`),
      Buffer.from(shot.data, "base64"),
    );
    await evaluate(`document.getElementById('learnBackBtn').click()`);
    await until(`!!document.getElementById('learnSearchInput')`);
    await evaluate(
      `const input=document.getElementById('learnSearchInput'); Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, 'value').set.call(input,'mushrooms'); input.dispatchEvent(new Event('input',{bubbles:true}));`,
    );
    await until(`!!document.getElementById('learnEmptyState')`);
    assert.equal(
      await evaluate(
        `new URL(document.querySelector('#learnEmptyState a').href).searchParams.get('topic')`,
      ),
      "mushrooms",
    );
    await call("Input.dispatchKeyEvent", {
      type: "keyDown",
      key: "Escape",
      code: "Escape",
      windowsVirtualKeyCode: 27,
    });
    await call("Input.dispatchKeyEvent", {
      type: "keyUp",
      key: "Escape",
      code: "Escape",
      windowsVirtualKeyCode: 27,
    });
    await until(`!document.getElementById('learnModal').open`);
    assert.equal(await evaluate(`document.activeElement.id`), "openLearnHub");
    const pageShot = await call("Page.captureScreenshot", { format: "png" });
    await writeFile(
      resolve(directory, `public-learning-${width}.png`),
      Buffer.from(pageShot.data, "base64"),
    );
  }
  await navigate("TestReactPages-dashboard-html.html");
  assert.equal(
    await evaluate(`!!document.querySelector('nav a[href="/learning"]')`),
    true,
    "Dashboard needs a direct learning link",
  );
  await evaluate(`document.getElementById('openLearnHub').click()`);
  await until(`document.getElementById('learnModal').open`);
  assert.equal(
    await evaluate(`document.querySelectorAll('.learn-topic-card').length`),
    8,
  );
  await evaluate(`document.getElementById('learnModalClose').click()`);
  await until(`!document.getElementById('learnModal').open`);
  await call("Page.navigate", { url: baseURL + "/learning?topic=Beekeeping" });
  await until(
    `document.getElementById('custom-farming-type')?.value === 'Beekeeping'`,
  );
  assert.equal(
    await evaluate(`document.getElementById('farming-type').value`),
    "other",
  );
  await evaluate(
    `document.querySelector('.learning-starters button').click(); window.holdLearning=true; document.getElementById('learning-chat-form').requestSubmit();`,
  );
  await until(`!!window.releaseLearning`);
  assert.equal(
    await evaluate(
      `document.querySelectorAll('.learning-typing-dots i').length`,
    ),
    3,
  );
  assert.equal(
    await evaluate(
      `document.getElementById('clear-chat').disabled && document.getElementById('send-question').disabled`,
    ),
    true,
  );
  await evaluate(`window.holdLearning=false; window.releaseLearning();`);
  await until(`!document.getElementById('send-question').disabled`);
  assert.equal(
    await evaluate(
      `document.querySelector('#chat-messages strong').textContent`,
    ),
    "Healthy soil",
  );
  assert.equal(
    await evaluate(`document.querySelector('#chat-messages img')`),
    null,
    "AI text must not execute HTML",
  );
  await evaluate(
    `document.getElementById('learning-question').value='How often should I water?'; document.getElementById('learning-chat-form').requestSubmit();`,
  );
  await until(
    `window.testRequests.filter(r => r.path === '/learning/chat').length === 2 && !document.getElementById('send-question').disabled`,
  );
  const followUp = await evaluate(
    `JSON.parse(window.testRequests.filter(r => r.path === '/learning/chat').at(-1).body)`,
  );
  assert.equal(followUp.farmingType, "Crop farming");
  assert.deepEqual(
    followUp.messages.map((message) => message.role),
    ["user", "assistant", "user"],
  );
  await evaluate(
    `document.getElementById('farming-type').value='Poultry farming'; document.getElementById('farming-type').dispatchEvent(new Event('change'));`,
  );
  assert.equal(
    await evaluate(
      `document.querySelectorAll('#chat-messages .learning-message').length`,
    ),
    1,
  );
  await evaluate(
    `window.failLearning=true; document.getElementById('learning-question').value='How do I house chickens?'; document.getElementById('learning-chat-form').requestSubmit();`,
  );
  await until(`!!document.querySelector('.learning-message-error')`);
  assert.equal(
    await evaluate(`document.getElementById('learning-question').value`),
    "How do I house chickens?",
  );
  await evaluate(
    `window.failLearning=false; document.getElementById('learning-chat-form').requestSubmit();`,
  );
  await until(
    `window.testRequests.filter(r => r.path === '/learning/chat').length === 4 && !document.getElementById('send-question').disabled`,
  );
  const retry = await evaluate(
    `JSON.parse(window.testRequests.filter(r => r.path === '/learning/chat').at(-1).body)`,
  );
  assert.equal(
    retry.messages.length,
    1,
    "Failed responses must not enter conversation history",
  );
  await evaluate(`document.getElementById('clear-chat').click()`);
  assert.equal(
    await evaluate(
      `document.querySelectorAll('#chat-messages .learning-message').length`,
    ),
    1,
  );
  await evaluate(
    `document.getElementById('learning-voice-start').click(); window.testRecognition.onresult({results:[[{transcript:'How do I plant maize?'}]]}); document.getElementById('learning-voice-stop').click();`,
  );
  assert.equal(
    await evaluate(`document.getElementById('learning-question').value`),
    "How do I plant maize?",
  );
  assert.equal(
    await evaluate(`document.getElementById('learning-read-aloud').checked`),
    true,
  );
  await evaluate(
    `document.getElementById('learning-chat-form').requestSubmit()`,
  );
  await until(
    `!document.getElementById('send-question').disabled && window.spokenAnswers.length > 0`,
  );
  assert.equal(
    await evaluate(`window.spokenAnswers.join(' ').includes('Healthy soil')`),
    true,
  );
  await evaluate(`document.getElementById('learning-speech-stop').click()`);
  assert.equal(
    await evaluate(`document.getElementById('learning-speech-stop').hidden`),
    true,
  );
  await evaluate(
    `window.failImages=true; document.querySelector('#chat-messages .ai-example-button').click()`,
  );
  await until(
    `document.querySelector('#chat-messages .example-image-status').textContent.includes('Please try again')`,
  );
  assert.equal(
    await evaluate(
      `!document.querySelector('#chat-messages .ai-example-button').disabled`,
    ),
    true,
  );
  await evaluate(
    `window.failImages=false; document.querySelector('#chat-messages .ai-example-button').click()`,
  );
  await until(
    `!!document.querySelector('#chat-messages .ai-example-picture img')`,
  );
  assert.equal(
    await evaluate(
      `document.querySelector('#chat-messages .ai-example-picture figcaption').textContent.includes('Example drawing')`,
    ),
    true,
  );
  await evaluate(
    `document.querySelector('#chat-messages .learning-message-assistant:last-child .ai-listen-button').click()`,
  );
  assert.equal(
    await evaluate(`!document.getElementById('learning-speech-stop').hidden`),
    true,
  );
  await evaluate(`document.getElementById('clear-chat').click()`);
  assert.equal(
    await evaluate(
      `document.getElementById('learning-speech-stop').hidden && !document.querySelector('#chat-messages .ai-example-picture')`,
    ),
    true,
  );

  // Exercise browser audio recording when voice typing is unavailable.
  await call("Page.navigate", { url: baseURL + "/learning?recorder=1" });
  await until(
    `!!document.getElementById('learning-voice-start') && !document.getElementById('learning-voice-start').disabled`,
  );
  await evaluate(`document.getElementById('learning-voice-start').click()`);
  await until(
    `document.getElementById('learning-voice-start').getAttribute('aria-pressed') === 'true'`,
  );
  await new Promise((resolve) => setTimeout(resolve, 700));
  await evaluate(
    `window.holdTranscription=true; document.getElementById('learning-voice-stop').click()`,
  );
  await until(`!!window.releaseTranscription`);
  assert.equal(
    await evaluate(`document.getElementById('learning-voice-start').disabled`),
    true,
  );
  await evaluate(
    `document.getElementById('learning-question').value='Old question'; document.getElementById('clear-chat').click(); window.releaseTranscription();`,
  );
  await until(`!document.getElementById('learning-voice-start').disabled`);
  assert.equal(
    await evaluate(`document.getElementById('learning-question').value`),
    "Old question",
    "A discarded transcription must not return into the chat",
  );
  await evaluate(
    `window.holdTranscription=false; document.getElementById('learning-voice-start').click()`,
  );
  await until(
    `document.getElementById('learning-voice-start').getAttribute('aria-pressed') === 'true'`,
  );
  await new Promise((resolve) => setTimeout(resolve, 700));
  await evaluate(`document.getElementById('learning-voice-stop').click()`);
  await until(
    `document.getElementById('learning-question').value === 'How do I plant maize?'`,
  );
  assert.equal(
    await evaluate(
      `window.testRequests.find(r=>r.path==='/learning/transcribe').body.audio.size > 0`,
    ),
    true,
  );

  await navigate("TestReactPages-ai-assistant-html.html");
  assert.equal(
    await evaluate(`document.querySelectorAll('#category option').length`),
    20,
  );
  await evaluate(
    `window.failDiagnosis=true; document.getElementById('category').value='Plant Problems'; document.getElementById('description').value='My leaves are yellow'; document.getElementById('ai-form').requestSubmit()`,
  );
  await until(
    `document.getElementById('submit-status').textContent.includes('Please try again')`,
  );
  assert.equal(
    await evaluate(`document.getElementById('description').value`),
    "My leaves are yellow",
  );
  assert.equal(
    await evaluate(`document.querySelector('.send-button').disabled`),
    false,
  );
  await evaluate(
    `window.failDiagnosis=false; document.getElementById('diagnosis-read-aloud').checked=true; document.getElementById('ai-form').requestSubmit()`,
  );
  await until(
    `document.getElementById('ai-result-text').textContent.startsWith('Check the soil.')`,
  );
  assert.equal(
    await evaluate(`document.querySelector('#ai-result-text img')`),
    null,
  );
  assert.equal(await evaluate(`window.spokenAnswers.length > 0`), true);
  await evaluate(
    `document.querySelector('#ai-result-tools .ai-example-button').click()`,
  );
  await until(
    `!!document.querySelector('#ai-result-tools .ai-example-picture')`,
  );
  assert.deepEqual(
    await evaluate(
      `JSON.parse(window.testRequests.find(r=>r.path==='/ai-assistant/example').body)`,
    ),
    { diagnosis_id: 42 },
  );
  await evaluate(`document.getElementById('start-audio').click()`);
  await until(`!document.getElementById('stop-audio').hidden`);
  await evaluate(`document.getElementById('ai-form').requestSubmit()`);
  assert.equal(
    await evaluate(
      `document.getElementById('submit-status').textContent.includes('Finish recording')`,
    ),
    true,
  );
  await new Promise((resolve) => setTimeout(resolve, 700));
  await evaluate(`document.getElementById('stop-audio').click()`);
  await until(`!document.getElementById('audio-preview').hidden`);
  await evaluate(
    `window.DataTransfer=undefined; window.holdDiagnosis=true; document.getElementById('ai-form').requestSubmit()`,
  );
  await until(`!!window.releaseDiagnosis`);
  assert.equal(
    await evaluate(
      `window.testRequests.filter(r=>r.path==='/ai-assistant').at(-1).body.audio.size > 0`,
    ),
    true,
    "Captured audio must reach diagnosis without assigning input.files",
  );
  assert.equal(
    await evaluate(`document.getElementById('start-audio').disabled`),
    true,
  );
  await evaluate(`window.holdDiagnosis=false; window.releaseDiagnosis()`);
  await until(`!document.querySelector('.send-button').disabled`);
  await evaluate(`document.getElementById('remove-audio').click()`);
  assert.equal(
    await evaluate(`document.getElementById('audio-preview').hidden`),
    true,
  );
  const adviceShot = await call("Page.captureScreenshot", { format: "png" });
  await writeFile(
    resolve(directory, "diagnosis-voice-pictures.png"),
    Buffer.from(adviceShot.data, "base64"),
  );
  console.log(
    "Learning, dashboard hub, topic handoff, voice questions, spoken replies, pictures, diagnosis, retries, and mobile layouts passed.",
  );
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
  const selfieHelp = await evaluate(
    `document.querySelector('.profile-section .field-help').textContent.replace(/\\s+/g, ' ').trim()`,
  );
  assert.match(
    selfieHelp,
    /selfie is required and will become your public profile picture/,
  );
  assert.match(selfieHelp, /private original.*compare with withdrawal selfies/);
  assert.match(selfieHelp, /even if you change your profile picture later/);
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
    true,
    "A confirmed phone and PIN must still require a selfie",
  );
  await evaluate(`document.getElementById('register-form').requestSubmit()`);
  await until(
    `document.querySelector('.message').textContent.includes('Take a selfie')`,
  );
  assert.equal(
    await evaluate(
      `window.testRequests.some(request => request.path === '/register')`,
    ),
    false,
  );
  await evaluate(`document.getElementById('open-selfie-camera').click()`);
  await until(`!document.getElementById('take-selfie').disabled`);
  await evaluate(`document.getElementById('take-selfie').click()`);
  await until(`!document.getElementById('selfie-preview').hidden`);
  const selfieStatus = await evaluate(
    `document.getElementById('selfie-status').textContent`,
  );
  assert.match(selfieStatus, /public profile picture/);
  assert.match(selfieStatus, /private original.*withdrawal verification/);
  assert.equal(
    await evaluate(`document.getElementById('complete-registration').disabled`),
    false,
  );
  await evaluate(`document.getElementById('register-form').requestSubmit()`);
  await until(`location.pathname === '/login'`);
  const registered = await evaluate(
    `JSON.parse(sessionStorage.getItem('registrationRequest'))`,
  );
  assert.equal(registered.profile_picture.type, "image/jpeg");
  assert.ok(
    registered.profile_picture.size > 0 &&
      registered.profile_picture.size <= 2 * 1024 * 1024,
  );
  assert.equal(registered.transaction_pin, "582491");
  console.log(
    "Login validation, password visibility, required profile selfie and private-original disclosure, phone confirmation, PIN gating, multipart registration and redirect passed.",
  );
  await navigate("TestReactPages-storage-html.html");
  await evaluate(`document.getElementById('add-btn').click()`);
  await until(
    `document.getElementById('modal').classList.contains('modal-open') && document.getElementById('form-action').value === 'create'`,
  );
  assert.equal(
    await evaluate(`document.getElementById('product-id').value`),
    "",
  );
  assert.equal(await evaluate(`document.getElementById('produce').value`), "");
  await evaluate(`document.getElementById('capture-location').click()`);
  await until(
    `document.getElementById('location-status').textContent.includes('Allow location access')`,
  );
  await evaluate(`document.getElementById('close-btn').click()`);
  await until(
    `!document.getElementById('modal').classList.contains('modal-open')`,
  );
  await navigate("TestReactPages-marketplace-html.html");
  await evaluate(
    `const input=document.getElementById('search');Object.getOwnPropertyDescriptor(HTMLInputElement.prototype,'value').set.call(input,'no matching product');input.dispatchEvent(new Event('input',{bubbles:true}));`,
  );
  await until(
    `document.querySelectorAll('#product-grid .product-card').length === 0`,
  );
  await navigate("TestReactPages-profile-html.html");
  await evaluate(`document.getElementById('edit-profile-btn').click()`);
  await until(
    `getComputedStyle(document.getElementById('edit-modal')).display === 'flex'`,
  );
  await evaluate(`document.getElementById('cancel-edit-btn').click()`);
  await until(
    `getComputedStyle(document.getElementById('edit-modal')).display === 'none'`,
  );
  await navigate("TestReactPages-wallet-html.html");
  await evaluate(`document.getElementById('wallet-balance-toggle').click()`);
  await until(
    `document.getElementById('wallet-balance-value').textContent === '••••••'`,
  );
  await evaluate(
    `document.querySelector('[data-wallet-finish="silver"]').click()`,
  );
  await until(
    `document.getElementById('wallet-metal-card').dataset.finish === 'silver'`,
  );
  console.log(
    "Storage editor and GPS denial, marketplace search, profile editing, and wallet controls passed.",
  );
  assert.deepEqual(exceptions, [], "Browser JavaScript exceptions");
  assert.deepEqual(overflows, [], "Pages overflow their viewport");
  console.log("PASS: 70 page layouts and interactive authentication flows.");
  await send("Browser.close");
} finally {
  clearTimeout(timeout);
  browser?.close();
  chrome.kill();
  await new Promise((resolve) => server.close(resolve));
}
