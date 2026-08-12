const images = [

    "/static/assets/images/Farmer1.jpeg",
    "/static/assets/images/Farmer2.jpeg",
    "/static/assets/images/Farmer3.jpeg",
    "/static/assets/images/Farmer4.jpeg",
    "/static/assets/images/Farmer5.jpeg",
    "/static/assets/images/Farmer5.jpeg",
    "/static/assets/images/Farmer6.jpeg",
    "/static/assets/images/Farmer7.jpeg",
    "/static/assets/images/Farmer8.jpeg",
    "/static/assets/images/Farmer9.jpeg",
    "/static/assets/images/Farmer10.jpeg"

];

const slider = document.getElementById("hero-slider");

let currentImage = 0;

function changeImage(){

    slider.style.opacity = 0;

    setTimeout(() => {

        currentImage++;

        if(currentImage >= images.length){

            currentImage = 0;

        }

        slider.src = images[currentImage];

        slider.style.opacity = 1;

    },400);

}

setInterval(changeImage,4000);