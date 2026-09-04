document.addEventListener("DOMContentLoaded", () => {
    /* ============================================================
       ELEMENTS
       ============================================================ */
    const cartContainer = document.querySelector(".cart-container");
    const cartItemsWrapper = document.querySelector(".cart-items");
    const removeForms = document.querySelectorAll(".remove-form");
    const cartSummary = document.querySelector(".cart-summary");

    /* ============================================================
       DYNAMIC SUMMARY & COUNT RECALCULATION
       ============================================================ */
    function updateCartSummary() {
        const remainingItems = Array.from(
            document.querySelectorAll(".cart-item")
        ).filter((item) => !item.hidden);

        const itemCountElement = cartSummary?.querySelector("p:nth-of-type(1) strong");
        const totalAmountElement = cartSummary?.querySelector("p:nth-of-type(2) strong");

        // Update item count indicator
        if (itemCountElement) {
            itemCountElement.textContent = String(remainingItems.length);
        }

        // Recalculate total price sum across remaining active items
        if (totalAmountElement) {
            let total = 0;
            remainingItems.forEach((card) => {
                const priceText = card.querySelector(".total-price strong")?.textContent || "";
                // Extract clean numerical digits from formatted currency (e.g., ₦150,000 -> 150000)
                const numericPrice = parseFloat(priceText.replace(/[^0-9.]/g, "")) || 0;
                total += numericPrice;
            });

            // Format as Nigerian Naira with thousands separators
            const formattedTotal = new Intl.NumberFormat("en-NG", {
                style: "currency",
                currency: "NGN",
                maximumFractionDigits: 2,
            }).format(total).replace("NGN", "₦").trim();

            totalAmountElement.textContent = formattedTotal;
        }

        // Render empty state if all items are removed
        if (remainingItems.length === 0) {
            renderEmptyCartState();
        }
    }

    /* ============================================================
       EMPTY CART DOM SWAP
       ============================================================ */
    function renderEmptyCartState() {
        if (!cartContainer) return;

        cartContainer.innerHTML = `
            <section class="empty-cart">
                <h2>Your cart is empty</h2>
                <p>When you accept a negotiated offer, the product will appear here.</p>
                <a href="/marketplace" class="marketplace-button">
                    Go to Marketplace
                </a>
            </section>
        `;
    }

    /* ============================================================
       REMOVAL FORM CONFIRMATION & AJAX DELETION
       ============================================================ */
    removeForms.forEach((form) => {
        form.addEventListener("submit", async (event) => {
            const card = form.closest(".cart-item");
            const productName = card?.querySelector(".cart-product h2")?.textContent.trim() || "this item";

            const confirmed = window.confirm(
                `Are you sure you want to remove "${productName}" from your cart?`
            );

            if (!confirmed) {
                event.preventDefault();
                return;
            }

            // Optional enhancement: Submit via fetch to bypass hard page reload
            /*
            event.preventDefault();
            const submitBtn = form.querySelector(".remove-button");
            
            try {
                if (submitBtn) submitBtn.disabled = true;
                const formData = new FormData(form);

                const response = await fetch(form.action, {
                    method: "POST",
                    body: formData,
                    headers: {
                        "X-Requested-With": "XMLHttpRequest",
                    },
                });

                if (response.ok) {
                    // Smooth visual fade and remove card
                    card.style.transition = "opacity 0.3s ease, transform 0.3s ease";
                    card.style.opacity = "0";
                    card.style.transform = "scale(0.95)";

                    setTimeout(() => {
                        card.remove();
                        updateCartSummary();
                    }, 300);
                } else {
                    alert("Failed to remove item. Please try again.");
                    if (submitBtn) submitBtn.disabled = false;
                }
            } catch (err) {
                console.error("Cart removal error:", err);
                if (submitBtn) submitBtn.disabled = false;
            }
            */
        });
    });
});