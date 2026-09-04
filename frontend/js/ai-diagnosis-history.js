document.addEventListener("DOMContentLoaded", () => {
    /* ============================================================
       ELEMENTS
       ============================================================ */

    const searchInput = document.getElementById("history-search");
    const categoryFilter = document.getElementById("category-filter");
    const resultCount = document.getElementById("result-count");
    const diagnosisList = document.getElementById("diagnosis-list");
    const cards = Array.from(document.querySelectorAll(".diagnosis-card"));
    const noResults = document.getElementById("no-search-results");

    // Modal Elements
    const modal = document.getElementById("result-modal");
    const modalOverlay = document.getElementById("modal-overlay");
    const modalClose = document.getElementById("modal-close");
    const modalTitle = document.getElementById("modal-title");
    const modalDate = document.getElementById("modal-date");
    const modalCategory = document.getElementById("modal-category");
    const modalDescription = document.getElementById("modal-description");
    const modalResult = document.getElementById("modal-result");

    let lastFocusedElement = null;


    /* ============================================================
       SEARCH & CATEGORY FILTERING
       ============================================================ */

    function filterDiagnoses() {
        if (!cards.length) {
            if (resultCount) resultCount.textContent = "0";
            return;
        }

        const query = searchInput ? searchInput.value.trim().toLowerCase() : "";
        const selectedCategory = categoryFilter ? categoryFilter.value : "all";
        let visibleCount = 0;

        cards.forEach((card) => {
            const cardCategory = card.getAttribute("data-category") || "";
            const searchContext = (card.getAttribute("data-search") || "").toLowerCase();

            // Category match check
            const matchesCategory =
                selectedCategory === "all" ||
                cardCategory.toLowerCase() === selectedCategory.toLowerCase();

            // Search query match check
            const matchesSearch =
                query === "" || searchContext.includes(query);

            // Toggle card visibility
            if (matchesCategory && matchesSearch) {
                card.hidden = false;
                visibleCount++;
            } else {
                card.hidden = true;
            }
        });

        // Update result counter in UI
        if (resultCount) {
            resultCount.textContent = String(visibleCount);
        }

        // Display or hide the "No diagnoses found" search state
        if (noResults) {
            noResults.hidden = visibleCount > 0;
        }
    }

    searchInput?.addEventListener("input", filterDiagnoses);
    categoryFilter?.addEventListener("change", filterDiagnoses);

    // Initial count calculation on page load
    filterDiagnoses();


    /* ============================================================
       MODAL INSPECTION
       ============================================================ */

    function openModal(card) {
        if (!modal || !card) {
            return;
        }

        lastFocusedElement = document.activeElement;

        // Extract information directly from the targeted card DOM elements
        const cardId = card.querySelector(".diagnosis-id")?.textContent.trim() || "";
        const categoryHeading = card.querySelector(".diagnosis-category h2")?.textContent.trim() || "N/A";
        const dateFormatted = card.querySelector(".diagnosis-card-header time")?.textContent.trim() || "";
        const descriptionText = card.querySelector(".diagnosis-description p")?.textContent.trim() || "No description provided.";
        const resultHTML = card.querySelector(".result-content")?.innerHTML || "";

        // Populate modal sections
        if (modalTitle) modalTitle.textContent = `Diagnosis ${cardId}`;
        if (modalDate) modalDate.textContent = dateFormatted;
        if (modalCategory) modalCategory.textContent = categoryHeading;
        if (modalDescription) modalDescription.textContent = descriptionText;
        if (modalResult) modalResult.innerHTML = resultHTML;

        // Show modal and lock background scrolling
        modal.hidden = false;
        modal.setAttribute("aria-hidden", "false");
        document.body.style.overflow = "hidden";

        if (modalClose) {
            modalClose.focus();
        }
    }

    function closeModal() {
        if (!modal) {
            return;
        }

        modal.hidden = true;
        modal.setAttribute("aria-hidden", "true");
        document.body.style.overflow = "";

        // Restore keyboard focus to the triggering button
        if (lastFocusedElement) {
            lastFocusedElement.focus();
        }
    }

    // Event delegation for "View Full Result" buttons
    diagnosisList?.addEventListener("click", (event) => {
        const viewBtn = event.target.closest(".view-result-button");

        if (viewBtn) {
            const card = viewBtn.closest(".diagnosis-card");
            openModal(card);
        }
    });

    modalClose?.addEventListener("click", closeModal);
    modalOverlay?.addEventListener("click", closeModal);

    // Close modal on Escape key press
    document.addEventListener("keydown", (event) => {
        if (event.key === "Escape" && modal && !modal.hidden) {
            closeModal();
        }
    });


    /* ============================================================
       DELETE FORM CONFIRMATION
       ============================================================ */

    const deleteForms = document.querySelectorAll(".delete-diagnosis-form");

    deleteForms.forEach((form) => {
        form.addEventListener("submit", (event) => {
            const card = form.closest(".diagnosis-card");
            const diagnosisId = card?.querySelector(".diagnosis-id")?.textContent.trim() || "";

            const confirmed = window.confirm(
                `Are you sure you want to delete diagnosis ${diagnosisId}? This record will be permanently removed.`
            );

            if (!confirmed) {
                event.preventDefault();
            }
        });
    });
});