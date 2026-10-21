const partnerState = document.getElementById("partner-state");
const partnerLGA = document.getElementById("partner-lga");

async function loadPartnerStates() {
    if (!partnerState || !partnerLGA) {
        return;
    }

    try {
        const response = await fetch("/api/states");
        if (!response.ok) {
            throw new Error("Failed to load states");
        }
        const states = await response.json();
        states.forEach((item) => {
            const option = document.createElement("option");
            option.value = item.id;
            option.textContent = item.name;
            partnerState.appendChild(option);
        });
    } catch (error) {
        console.error("Partner state loading error:", error);
    }
}

async function loadPartnerLGAs() {
    const stateID = partnerState.value;
    partnerLGA.innerHTML = '<option value="">Select your LGA</option>';
    partnerLGA.disabled = true;

    if (!stateID) {
        return;
    }

    try {
        const response = await fetch(`/api/states/${stateID}/lgas`);
        if (!response.ok) {
            throw new Error("Failed to load LGAs");
        }
        const lgas = await response.json();
        lgas.forEach((item) => {
            const option = document.createElement("option");
            option.value = item.id;
            option.textContent = item.name;
            partnerLGA.appendChild(option);
        });
        partnerLGA.disabled = false;
    } catch (error) {
        console.error("Partner LGA loading error:", error);
    }
}

if (partnerState && partnerLGA) {
    partnerState.addEventListener("change", loadPartnerLGAs);
    loadPartnerStates();
}
