/* =========================================
   DASHBOARD COUNTERS
========================================= */

const counters = document.querySelectorAll(".counter");


counters.forEach(counter => {

    const target = Number(counter.dataset.target);

    let count = 0;

    const speed = Math.max(
        1,
        Math.ceil(target / 100)
    );


    function updateCounter() {

        if (count < target) {

            count += speed;

            if (count > target) {

                count = target;

            }

            counter.textContent =
                count.toLocaleString();

            requestAnimationFrame(
                updateCounter
            );

        } else {

            counter.textContent =
                target.toLocaleString();

        }

    }


    updateCounter();

});



/* =========================================
   GREETING
========================================= */

const greeting =
    document.getElementById("greeting");


if (greeting) {

    const hour =
        new Date().getHours();


    if (hour < 12) {

        greeting.textContent =
            "Good Morning";

    }

    else if (hour < 17) {

        greeting.textContent =
            "Good Afternoon";

    }

    else {

        greeting.textContent =
            "Good Evening";

    }

}



/* =========================================
   CURRENT DATE
========================================= */

const today =
    document.getElementById("today");


if (today) {

    today.textContent =
        new Date().toDateString();

}



/* =========================================
   PROFILE DROPDOWN
========================================= */

const profileMenu =
    document.querySelector(".profile-menu");


const profileTrigger =
    document.getElementById("profile-arrow");


const profileDropdown =
    document.getElementById("profile-dropdown");



if (
    profileMenu &&
    profileTrigger &&
    profileDropdown
) {


    /* -----------------------------------------
       OPEN / CLOSE MENU
    ----------------------------------------- */

    profileTrigger.addEventListener(
        "click",
        function (event) {

            event.stopPropagation();


            const isOpen =
                profileMenu.classList.toggle(
                    "open"
                );


            profileTrigger.setAttribute(
                "aria-expanded",
                String(isOpen)
            );

        }
    );



    /* -----------------------------------------
       PREVENT DROPDOWN CLICK FROM PROPAGATING
    ----------------------------------------- */

    profileDropdown.addEventListener(
        "click",
        function (event) {

            event.stopPropagation();

        }
    );



    /* -----------------------------------------
       CLOSE WHEN CLICKING OUTSIDE
    ----------------------------------------- */

    document.addEventListener(
        "click",
        function () {

            profileMenu.classList.remove(
                "open"
            );


            profileTrigger.setAttribute(
                "aria-expanded",
                "false"
            );

        }
    );



    /* -----------------------------------------
       CLOSE WITH ESCAPE
    ----------------------------------------- */

    document.addEventListener(
        "keydown",
        function (event) {

            if (event.key === "Escape") {

                profileMenu.classList.remove(
                    "open"
                );


                profileTrigger.setAttribute(
                    "aria-expanded",
                    "false"
                );

            }

        }
    );

}