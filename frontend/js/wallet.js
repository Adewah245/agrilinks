document.addEventListener("DOMContentLoaded", async () => {
    const select = document.getElementById("wallet-bank");
    if (!select) return;
    try {
        const response = await fetch("/wallet/banks", { headers: { Accept: "application/json" } });
        if (!response.ok || response.redirected) throw new Error("Could not load the bank list. Please reload the page.");
        const banks = await response.json();
        select.replaceChildren(new Option("Choose your bank", ""));
        banks.forEach(bank => select.add(new Option(bank.name, bank.code)));
    } catch (error) {
        select.replaceChildren(new Option("Bank list unavailable", ""));
        document.getElementById("bank-error").textContent = error.message;
    }
});
