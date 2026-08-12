const search = document.getElementById("search");
const grid = document.getElementById("product-grid");

if (search && grid) {
    search.addEventListener("keyup", function () {
        const keyword = this.value.toLowerCase();
        const cards = grid.querySelectorAll(".product-card");

        cards.forEach(card => {
            const name = card.querySelector("h3")?.textContent.toLowerCase() || "";
            card.style.display = name.includes(keyword) ? "" : "none";
        });
    });
}
