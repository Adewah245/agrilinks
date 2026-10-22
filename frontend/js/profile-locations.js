(() => {
    const stateSelect = document.querySelector("[data-state-select]") || document.getElementById("profile-state");
    const lgaSelect = document.querySelector("[data-lga-select]") || document.getElementById("profile-lga");

    if (!stateSelect || !lgaSelect) {
        return;
    }

    const optionLabel = (value, fallback = "Choose an option") => {
        if (typeof value === "string" && value.trim() !== "") {
            return value.trim();
        }
        return fallback;
    };

    const setOptions = (select, items, prompt, selectedValue = "") => {
        const options = Array.isArray(items) ? items : [];
        select.innerHTML = "";
        const emptyOption = document.createElement("option");
        emptyOption.value = "";
        emptyOption.textContent = prompt;
        select.appendChild(emptyOption);

        options.forEach((item) => {
            const name = typeof item === "string" ? item : item.name;
            if (!name) {
                return;
            }
            const option = document.createElement("option");
            option.value = name;
            option.textContent = name;
            if (typeof item === "object" && item !== null) {
                option.dataset.stateId = item.id ?? "";
                option.dataset.lgaId = item.id ?? "";
            }
            select.appendChild(option);
        });

        if (selectedValue) {
            const match = [...select.options].find((option) => option.value === selectedValue);
            if (match) {
                select.value = match.value;
            }
        }
    };

    const stateIDForName = (name) => {
        const match = [...stateSelect.options].find((option) => option.value === name);
        return match ? match.dataset.stateId : "";
    };

    async function loadLGAs(stateName, selectedLGA = "") {
        const stateID = stateIDForName(stateName);
        lgaSelect.disabled = true;
        setOptions(lgaSelect, [], "Loading LGAs…");

        if (!stateID) {
            setOptions(lgaSelect, [], "Choose a state first");
            lgaSelect.disabled = true;
            return;
        }

        try {
            const response = await fetch(`/api/states/${stateID}/lgas`);
            if (!response.ok) {
                throw new Error("Failed to load LGAs");
            }
            const lgas = await response.json();
            setOptions(lgaSelect, lgas, "Select your LGA", selectedLGA);
            lgaSelect.disabled = false;
        } catch (error) {
            console.error("LGA load error:", error);
            setOptions(lgaSelect, [], "Unable to load LGAs");
            lgaSelect.disabled = true;
        }
    }

    async function loadStates() {
        try {
            const response = await fetch("/api/states");
            if (!response.ok) {
                throw new Error("Failed to load states");
            }
            const states = await response.json();
            const currentState = stateSelect.dataset.selected || stateSelect.value || "";
            setOptions(stateSelect, states, "Select your state", currentState);
            stateSelect.disabled = false;

            if (currentState) {
                const selectedLGA = lgaSelect.dataset.selected || lgaSelect.value || "";
                await loadLGAs(currentState, selectedLGA);
            } else {
                setOptions(lgaSelect, [], "Choose a state first");
                lgaSelect.disabled = true;
            }
        } catch (error) {
            console.error("State load error:", error);
            setOptions(stateSelect, [], "Unable to load states");
            stateSelect.disabled = true;
            setOptions(lgaSelect, [], "Unable to load LGAs");
            lgaSelect.disabled = true;
        }
    }

    stateSelect.addEventListener("change", () => {
        const selection = optionLabel(stateSelect.value, "");
        if (!selection) {
            setOptions(lgaSelect, [], "Choose a state first");
            lgaSelect.disabled = true;
            return;
        }
        loadLGAs(selection);
    });

    loadStates();
})();
