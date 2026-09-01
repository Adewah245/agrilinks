const registerForm = document.getElementById("register-form");


const firstName = document.getElementById("first-name");
const lastName = document.getElementById("last-name");
const email = document.getElementById("email");
const phone = document.getElementById("phone");

const password = document.getElementById("password");
const confirmPassword = document.getElementById("confirm-password");

const button = registerForm.querySelector("button");



// ===============================
// MESSAGE AREA
// ===============================

const message = document.createElement("p");

message.className = "message";

registerForm.appendChild(message);





// ===============================
// PASSWORD STRENGTH
// ===============================


const strength = document.createElement("small");

strength.className = "password-strength";

password.parentNode.appendChild(strength);



password.addEventListener("input",()=>{


    if(password.value.length < 8){

        strength.textContent="Weak Password";
        strength.style.color="red";

    }

    else if(password.value.length < 12){

        strength.textContent="Medium Password";
        strength.style.color="orange";

    }

    else{

        strength.textContent="Strong Password";
        strength.style.color="green";

    }


});







// ===============================
// SHOW / HIDE PASSWORD
// ===============================


const togglePassword =
document.getElementById("toggle-password");


const toggleConfirm =
document.getElementById("toggle-confirm-password");



togglePassword.addEventListener("click",()=>{


    if(password.type==="password"){

        password.type="text";

        togglePassword.classList.replace(
            "fa-eye",
            "fa-eye-slash"
        );

    }

    else{

        password.type="password";

        togglePassword.classList.replace(
            "fa-eye-slash",
            "fa-eye"
        );

    }


});






toggleConfirm.addEventListener("click",()=>{


    if(confirmPassword.type==="password"){

        confirmPassword.type="text";


        toggleConfirm.classList.replace(
            "fa-eye",
            "fa-eye-slash"
        );

    }


    else{


        confirmPassword.type="password";


        toggleConfirm.classList.replace(
            "fa-eye-slash",
            "fa-eye"
        );

    }


});









// ===============================
// FORM VALIDATION
// ===============================


registerForm.addEventListener("submit",(event)=>{


message.textContent="";





const role =
document.querySelector(
'input[name="role"]:checked'
).value;





// BASIC CHECK

if(
firstName.value.trim()==="" ||
lastName.value.trim()==="" ||
email.value.trim()==="" ||
phone.value.trim()===""
){


event.preventDefault();


message.textContent=
"Please complete all fields.";


message.style.color="red";


return;


}






// FARM TYPE CHECK


const farmTypes =
document.querySelectorAll(
'input[name="farm_type"]:checked'
);





if(
role==="farmer" &&
farmTypes.length===0
){


event.preventDefault();


message.textContent=
"Please select at least one farming type.";


message.style.color="red";


return;


}







// PASSWORD MATCH


if(password.value !== confirmPassword.value){


event.preventDefault();


message.textContent=
"Passwords do not match.";


message.style.color="red";


return;


}







if(password.value.length < 8){


event.preventDefault();


message.textContent=
"Password must be at least 8 characters.";


message.style.color="red";


return;


}







// ===============================
// PREPARE FARM DATA
// ===============================


if(role==="farmer"){


const selectedFarms =
Array.from(farmTypes)
.map(item=>item.value);



console.log(
"Selected farming types:",
selectedFarms
);


/*
Example output:

[
 "crop",
 "poultry",
 "fishery"
]

Later backend will save this.
*/


}






// SUCCESS

button.disabled=true;

button.textContent=
"Creating Account...";


});








// ===============================
// BACKGROUND IMAGE ROTATION
// ===============================


const registerImages = [

"/static/assets/images/login-image6.jpeg",

"/static/assets/images/login-image7.jpeg",

"/static/assets/images/login-image8.jpeg",

"/static/assets/images/un.jpeg"

];



const leftPanel =
document.querySelector(".auth-left");



let current=0;



function changeBackground(){


current++;


if(current >= registerImages.length){

current=0;

}



leftPanel.style.backgroundImage=`

linear-gradient(
rgba(0,100,0,.82),
rgba(0,100,0,.82)
),

url('${registerImages[current]}')

`;

}



changeBackground();


setInterval(
changeBackground,
4000
);