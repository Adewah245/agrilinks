import assert from "node:assert/strict";
import { writeFile } from "node:fs/promises";
import { resolve } from "node:path";

export async function checkBankVerification({ call, navigate, evaluate, until, directory }) {
  await call("Page.addScriptToEvaluateOnNewDocument", {
    source: `
      const originalBankFetch = window.fetch;
      window.bankTest = {requests:[], modes:{}, pending:{}, failDirectory:location.search.includes('banks_unavailable=1')};
      window.fetch = async (input, options = {}) => {
        const path = new URL(String(input), location.origin).pathname;
        const json = (value, status = 200) => new Response(JSON.stringify(value), {status, headers:{'Content-Type':'application/json'}});
        if (path === '/wallet/banks') {
          if (bankTest.failDirectory) return json({error:'Bank directory unavailable.'}, 503);
          return json([{name:'Example Bank',code:'001'},{name:'Farmers Microfinance Bank',code:'100004'}]);
        }
        if (path !== '/wallet/resolve-account') return originalBankFetch(input, options);
        const details = Object.fromEntries(new URLSearchParams(options.body));
        bankTest.requests.push({url:String(input),method:options.method,cache:options.cache,...details});
        const key = details.bank_code + ':' + details.account_number;
        const mode = bankTest.modes[key];
        // Deliberately ignore AbortSignal here to prove stale responses are ignored.
        if (mode === 'hold') return json(await new Promise(resolve => {bankTest.pending[key] = resolve;}));
        if (mode === 'unavailable') return json({error:'Bank verification is temporarily unavailable.'}, 503);
        if (mode === 'invalid') return json({error:'Check the bank and account number.'}, 422);
        if (mode === 'expired') return json({error:'Please sign in.'}, 401);
        return json({...details,bank_name:'Example Bank',
          account_number:mode === 'wrong-number' ? '9999999999' : details.account_number,
          account_name:mode === 'empty-name' ? '' : mode === 'mismatch' ? 'Another Owner' : 'Ada Farmer',
          matches_profile:mode !== 'mismatch'});
      };
    `,
  });
  const fill = (id, value) => evaluate(`
    const element = document.getElementById(${JSON.stringify(id)});
    Object.getOwnPropertyDescriptor(HTMLInputElement.prototype,'value').set.call(element,${JSON.stringify(value)});
    element.dispatchEvent(new Event('input',{bubbles:true}));
  `);
  const chooseBank = (value) => evaluate(`
    const element = document.getElementById('wallet-bank');
    element.value = ${JSON.stringify(value)};
    element.dispatchEvent(new Event('change',{bubbles:true}));
  `);
  const name = () => evaluate(`document.getElementById('wallet-account-name')?.textContent || ''`);
  const disabled = () => evaluate(`document.getElementById('wallet-save-bank').disabled`);
  const pause = () => evaluate(`new Promise(resolve => setTimeout(resolve, 500))`);
  const fixture = "TestReactPages-wallet-html.html";

  await navigate(fixture + "?banks_unavailable=1");
  await until(`document.getElementById('bank-error').textContent.includes('Bank directory unavailable')`);
  assert.equal(await disabled(), true);
  assert.equal(await evaluate(`document.getElementById('wallet-bank').disabled`), true);
  await evaluate(`bankTest.failDirectory=false; document.querySelector('.wallet-bank-retry').click()`);
  await until(`document.getElementById('wallet-bank').options.length === 3`);
  await evaluate(`document.getElementById('wallet-bank-details').open=true`);
  assert.equal(await evaluate(`document.getElementById('wallet-account-name')`), null, "No account name should appear before verification");
  await chooseBank("001");
  await fill("wallet-account-number", "012345678");
  await pause();
  assert.equal(await evaluate(`bankTest.requests.length`), 0, "Incomplete numbers must not trigger a lookup");
  assert.equal(await disabled(), true);
  await fill("wallet-account-number", "0123456789");
  await until(`document.getElementById('wallet-account-name')?.textContent === 'Ada Farmer'`);
  assert.equal(await disabled(), false);
  assert.equal(await evaluate(`document.getElementById('wallet-account-name').tagName`), "OUTPUT");
  assert.equal(await evaluate(`document.getElementById('wallet-account-name').isContentEditable`), false);
  assert.equal(await evaluate(`document.querySelector('input[name="account_name"]:not([type="hidden"])')`), null, "Users cannot type an account name");
  assert.deepEqual(await evaluate(`bankTest.requests[0]`), {
    url: "/wallet/resolve-account", method: "POST", cache: "no-store",
    bank_code: "001", account_number: "0123456789",
  }, "Lookup must preserve leading zeros and exclude the PIN");
  await fill("wallet-bank-pin", "582491");

  await evaluate(`bankTest.modes['100004:0123456789']='mismatch'`);
  await chooseBank("100004");
  assert.equal(await name(), "", "Changing the bank must immediately clear the previous name");
  assert.equal(await disabled(), true);
  assert.equal(await evaluate(`document.getElementById('wallet-bank-pin').value`), "");
  await until(`document.getElementById('wallet-account-name')?.textContent === 'Another Owner'`);
  assert.equal(await disabled(), true);
  assert.equal(await evaluate(`document.getElementById('bank-error').textContent.includes('does not match')`), true);
  assert.equal(await evaluate(`document.querySelector('[data-bank-account-form]').dispatchEvent(new Event('submit',{bubbles:true,cancelable:true}))`), false, "An unmatched account must not submit");
  assert.notEqual(await evaluate(`document.querySelector('[data-bank-account-form]').dataset.submitting`), "true");

  await chooseBank("001");
  for (const [mode, message] of [
    ["invalid", "Check the bank and account number"],
    ["unavailable", "temporarily unavailable"],
    ["wrong-number", "did not confirm"],
    ["empty-name", "did not confirm"],
    ["expired", "session has expired"],
  ]) {
    await evaluate(`bankTest.modes['001:3333333333']=${JSON.stringify(mode)}`);
    await fill("wallet-account-number", "333333333");
    await fill("wallet-account-number", "3333333333");
    await until(`document.getElementById('bank-error').textContent.includes(${JSON.stringify(message)})`);
    assert.equal(await disabled(), true, mode);
    assert.equal(await name(), "", mode);
    await evaluate(`delete bankTest.modes['001:3333333333']; document.querySelector('.wallet-bank-retry').click()`);
    await until(`document.getElementById('wallet-account-name')?.textContent === 'Ada Farmer'`);
    assert.equal(await disabled(), false, "Retry must recover verification");
  }

  await evaluate(`bankTest.modes['001:1111111111']='hold'`);
  await fill("wallet-account-number", "1111111111");
  await until(`!!bankTest.pending['001:1111111111']`);
  await fill("wallet-account-number", "2222222222");
  assert.equal(await name(), "");
  assert.equal(await disabled(), true);
  await until(`document.getElementById('wallet-account-name')?.textContent === 'Ada Farmer'`);
  await evaluate(`bankTest.pending['001:1111111111']({bank_code:'001',account_number:'1111111111',account_name:'Stale Account',matches_profile:true})`);
  await pause();
  assert.equal(await name(), "Ada Farmer", "A late response must not replace the current verification");
  assert.equal(await evaluate(`document.getElementById('wallet-account-number').value`), "2222222222");

  await fill("wallet-bank-pin", "582491");
  await evaluate(`window.dispatchEvent(new PageTransitionEvent('pageshow',{persisted:true}))`);
  await until(`document.getElementById('wallet-save-bank').disabled && !document.getElementById('wallet-account-name')`);
  assert.equal(await evaluate(`document.getElementById('wallet-bank-pin').value`), "", "Browser history must not restore the PIN");
  await until(`document.getElementById('wallet-account-name')?.textContent === 'Ada Farmer'`);

  for (const width of [390, 1440]) {
    await call("Emulation.setDeviceMetricsOverride", {width,height:960,deviceScaleFactor:1,mobile:width < 500});
    await evaluate(`document.querySelectorAll('.wallet-move-panel').forEach(panel=>panel.open=panel.id==='wallet-bank-details');document.getElementById('wallet-bank-details').scrollIntoView({block:'start'})`);
    assert.equal(await evaluate(`document.documentElement.scrollWidth <= ${width + 1}`), true, "Bank verification must fit the viewport");
    const shot = await call("Page.captureScreenshot", {format:"png"});
    await writeFile(resolve(directory, `bank-verification-${width}.png`), Buffer.from(shot.data, "base64"));
  }

  await fill("wallet-bank-pin", "582491");
  await evaluate(`
    document.addEventListener('submit',event=>{
      window.submittedBank=Object.fromEntries(new FormData(event.target));
      event.preventDefault();
    },{once:true});
    document.querySelector('[data-bank-account-form]').requestSubmit();
  `);
  assert.deepEqual(await evaluate(`window.submittedBank`), {
    action: "save_bank", bank_code: "001", account_number: "2222222222",
    account_name: "Ada Farmer", transaction_pin: "582491",
  }, "Submitting must retain the verified details and PIN");
  assert.equal(await disabled(), true, "Prevent duplicate saves");
  console.log("Bank verification: name lookup, leading zeros, ownership mismatch, failed providers, retries, stale responses, browser history, mobile layout, and save submission passed.");
}
