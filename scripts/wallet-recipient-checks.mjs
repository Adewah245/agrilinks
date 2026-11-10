import assert from "node:assert/strict";

export async function checkWalletRecipient({
  call,
  navigate,
  evaluate,
  until,
}) {
  await call("Page.addScriptToEvaluateOnNewDocument", {
    source: `
      const originalRecipientFetch = window.fetch;
      window.recipientTest = {requests:[], modes:{}, pending:{}};
      window.fetch = async (input, options = {}) => {
        if (new URL(String(input), location.origin).pathname !== '/wallet/recipient')
          return originalRecipientFetch(input, options);
        const details = Object.fromEntries(new URLSearchParams(options.body));
        recipientTest.requests.push({method:options.method,cache:options.cache,...details});
        const mode = recipientTest.modes[details.recipient];
        const json = (value, status = 200) => new Response(JSON.stringify(value), {status,headers:{'Content-Type':'application/json'}});
        if (mode === 'hold') return json(await new Promise(resolve => {recipientTest.pending[details.recipient] = resolve;}));
        if (mode === 'missing') return json({error:'No active wallet found.'},404);
        if (mode === 'unavailable') return json({error:'Verification is temporarily unavailable.'},503);
        if (mode === 'expired') return json({error:'Please sign in.'},401);
        return json({user_id:2,full_name:mode === 'empty-name' ? '' : 'Bola Farmer',
          wallet_number:mode === 'empty-wallet' ? '' : 'AGW-U-2-NGN',
          currency:mode === 'wrong-currency' ? 'USD' : 'NGN',photo_url:''});
      };
    `,
  });
  const fill = (id, value) =>
    evaluate(`
    const input = document.getElementById(${JSON.stringify(id)});
    Object.getOwnPropertyDescriptor(HTMLInputElement.prototype,'value').set.call(input,${JSON.stringify(value)});
    input.dispatchEvent(new Event('input',{bubbles:true}));
  `);
  const name = () =>
    evaluate(
      `document.getElementById('wallet-recipient-name')?.textContent || ''`,
    );
  const disabled = () =>
    evaluate(
      `document.querySelector('[data-wallet-transfer-form] button[type="submit"]').disabled`,
    );
  const pause = () =>
    evaluate(`new Promise(resolve => setTimeout(resolve,500))`);
  await navigate("TestReactPages-wallet-html.html");
  await evaluate(`document.getElementById('wallet-transfer').open=true`);
  assert.equal(await name(), "");
  assert.equal(await disabled(), true);
  assert.equal(
    await evaluate(
      `document.querySelector('[data-wallet-transfer-form]').dispatchEvent(new Event('submit',{bubbles:true,cancelable:true}))`,
    ),
    false,
    "Verification is required even if submit is triggered directly",
  );
  assert.notEqual(
    await evaluate(
      `document.querySelector('[data-wallet-transfer-form]').dataset.submitting`,
    ),
    "true",
  );
  const requestKey = await evaluate(
    `document.querySelector('[data-wallet-transfer-form] [name="request_key"]').value`,
  );
  await fill("wallet-recipient", "AGW-U-2-NGN");
  await until(
    `document.getElementById('wallet-recipient-name')?.textContent === 'Bola Farmer'`,
  );
  assert.equal(await disabled(), false);
  assert.equal(
    await evaluate(`document.getElementById('wallet-recipient-name').tagName`),
    "OUTPUT",
  );
  assert.equal(
    await evaluate(
      `document.getElementById('wallet-recipient-name').isContentEditable`,
    ),
    false,
  );
  assert.equal(
    await evaluate(
      `document.querySelector('[data-wallet-transfer-form] [name="recipient_name"], [data-wallet-transfer-form] [name="account_name"]')`,
    ),
    null,
    "No user-supplied name is submitted",
  );
  assert.deepEqual(await evaluate(`recipientTest.requests[0]`), {
    method: "POST",
    cache: "no-store",
    recipient: "AGW-U-2-NGN",
  });
  for (const [mode, message] of [
    ["missing", "No active wallet"],
    ["unavailable", "temporarily unavailable"],
    ["expired", "session has expired"],
    ["empty-name", "couldn't confirm"],
    ["empty-wallet", "couldn't confirm"],
    ["wrong-currency", "couldn't confirm"],
  ]) {
    await evaluate(
      `recipientTest.modes['AGW-U-3-NGN']=${JSON.stringify(mode)}`,
    );
    await fill("wallet-recipient", "AGW-U-3-NG");
    await fill("wallet-recipient", "AGW-U-3-NGN");
    assert.equal(
      await name(),
      "",
      "Changing the recipient clears the name immediately",
    );
    assert.equal(await disabled(), true);
    await until(
      `document.getElementById('wallet-recipient-error').textContent.includes(${JSON.stringify(message)})`,
    );
    assert.equal(await name(), "");
    assert.equal(await disabled(), true);
    await evaluate(
      `delete recipientTest.modes['AGW-U-3-NGN'];document.querySelector('[data-wallet-transfer-form] .wallet-bank-retry').click()`,
    );
    await until(
      `document.getElementById('wallet-recipient-name')?.textContent === 'Bola Farmer'`,
    );
    assert.equal(await disabled(), false);
  }
  await evaluate(`recipientTest.modes['AGW-U-4-NGN']='hold'`);
  await fill("wallet-recipient", "AGW-U-4-NGN");
  await until(`!!recipientTest.pending['AGW-U-4-NGN']`);
  await fill("wallet-recipient", "AGW-U-2-NGN");
  await until(
    `document.getElementById('wallet-recipient-name')?.textContent === 'Bola Farmer'`,
  );
  await evaluate(
    `recipientTest.pending['AGW-U-4-NGN']({user_id:4,full_name:'Stale Name',wallet_number:'AGW-U-4-NGN',currency:'NGN'})`,
  );
  await pause();
  assert.equal(await name(), "Bola Farmer");
  assert.equal(
    await evaluate(
      `document.querySelector('[data-wallet-transfer-form] [name="recipient"]').value`,
    ),
    "AGW-U-2-NGN",
  );
  await evaluate(
    `window.dispatchEvent(new PageTransitionEvent('pageshow',{persisted:true}))`,
  );
  await until(`!document.getElementById('wallet-recipient-name')`);
  assert.equal(await disabled(), true);
  await until(
    `document.getElementById('wallet-recipient-name')?.textContent === 'Bola Farmer'`,
  );
  assert.equal(
    await evaluate(
      `document.querySelector('[data-wallet-transfer-form] [name="request_key"]').value`,
    ),
    requestKey,
    "Lookups must preserve the transaction request key",
  );
  await fill("wallet-recipient", "");
  assert.equal(await name(), "");
  assert.equal(await disabled(), true);
  await fill("wallet-recipient", "AGW-U-2-NGN");
  await until(
    `document.getElementById('wallet-recipient-name')?.textContent === 'Bola Farmer'`,
  );
  for (const width of [390, 1440]) {
    await call("Emulation.setDeviceMetricsOverride", {
      width,
      height: 960,
      deviceScaleFactor: 1,
      mobile: width < 500,
    });
    await evaluate(
      `document.querySelectorAll('.wallet-move-panel').forEach(panel=>panel.open=panel.id==='wallet-transfer');document.getElementById('wallet-transfer').scrollIntoView({block:'start'})`,
    );
    assert.equal(
      await evaluate(`document.documentElement.scrollWidth <= ${width + 1}`),
      true,
    );
  }
  await fill("wallet-transfer-amount", "100");
  await fill("wallet-transfer-code", "123456");
  await fill("wallet-transfer-pin", "582491");
  await evaluate(`
    document.addEventListener('submit',event=>{
      window.submittedTransfer=Object.fromEntries(new FormData(event.target));
      event.preventDefault();
    },{once:true});
    document.querySelector('[data-wallet-transfer-form]').requestSubmit();
  `);
  assert.equal(
    await evaluate(`window.submittedTransfer.recipient`),
    "AGW-U-2-NGN",
  );
  assert.equal(await disabled(), true);
  console.log(
    "Wallet recipient verification: names cannot be edited, failed and stale lookups cannot enable transfers, and verified wallet details are submitted.",
  );
}
