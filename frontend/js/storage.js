const modal = document.getElementById("modal");
const addBtn = document.getElementById("add-btn");
const closeBtn = document.getElementById("close-btn");
const search = document.getElementById("search");

addBtn.addEventListener("click", () => {
    modal.style.display = "flex";
});

closeBtn.addEventListener("click", () => {
    modal.style.display = "none";
});

modal.addEventListener("click", (event) => {
    if (event.target === modal) {
        modal.style.display = "none";
    }
});

search.addEventListener("keyup", function () {
    const keyword = this.value.toLowerCase();
    const rows = document.querySelectorAll("#storage-body tr");

    rows.forEach(row => {
        const produce = row.children[0]?.textContent.toLowerCase() || "";
        row.style.display = produce.includes(keyword) ? "" : "none";
    });
});
