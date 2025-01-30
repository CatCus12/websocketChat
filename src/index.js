
import "./defolt.css";
import "./style.css";
import Icon from './image/image.png';

let username = '';
const socket = new WebSocket("ws://localhost:8080/ws");
const send_button = document.getElementById("send");
const input = document.getElementById("message");
const entry = document.querySelector(".entry");

const chatSection = document.querySelector(".chat");
const messagesDiv = document.getElementById('messages');

const registrationForm = document.forms.register;
const usernameInput = registrationForm.elements.username;
const passwordInput = registrationForm.elements.password;
const authorizationForm = document.forms.authorization;
const usernameInputAuth = authorizationForm.elements.username;
const passwordInputAuth = authorizationForm.elements.password;

const entryFormRegistrationButton = document.querySelector('.entry-form__registration-button');
const entryFormAuthorizationButton = document.querySelector('.entry-form__authorization-button');

entryFormAuthorizationButton.addEventListener("click", showFormAuthorization);
entryFormRegistrationButton.addEventListener("click", showFormRegistration);

function showFormAuthorization(evt) {
    authorizationForm.classList.remove('hidden');
    registrationForm.classList.add('hidden');
    removeDisable(entryFormRegistrationButton);
    evt.target.setAttribute('disabled', true);
}

function showFormRegistration(evt) {
    authorizationForm.classList.add('hidden');
    registrationForm.classList.remove('hidden');
    evt.target.setAttribute('disabled', true);
    removeDisable(entryFormAuthorizationButton);
}

function removeDisable(element) {
    element.removeAttribute('disabled');
}

function handleFormSubmitAuth(isRegister){
    username = usernameInputAuth.value.trim();
    console.log('введенное имя',username);
    const password = passwordInputAuth.value.trim();
    console.log(password);
    if (username && password) {
        console.log(`${isRegister ? 'Регистрация' : 'Авторизация'}: ${username}, пароль: ${password}`);
        const authorizationObject = {
            name: username, 
            password: password, 
            is_register: isRegister 
        };
        const jsonAuthorization = JSON.stringify(authorizationObject);
        socket.send(jsonAuthorization);
    }
}

function handleFormSubmit(isRegister) {  
    username = usernameInput.value.trim();
    console.log('введенное имя',username);
    const password = passwordInput.value.trim();
    console.log(password);
    if (username && password) {
        console.log(`${isRegister ? 'Регистрация' : 'Авторизация'}: ${username}, пароль: ${password}`);
        const authorizationObject = {
            name: username, 
            password: password, 
            is_register: isRegister 
        };
        const jsonAuthorization = JSON.stringify(authorizationObject);
        socket.send(jsonAuthorization);
    }
}

// Для формы регистрации
registrationForm.addEventListener('submit', (e) => {
    e.preventDefault();
    handleFormSubmit(true); 
});
authorizationForm.addEventListener('submit', (e) => {
    e.preventDefault();
    handleFormSubmitAuth(false); 
});



let isAuthorized = false; 

socket.onmessage = function(event) {
    //const messagesDiv = document.getElementById("messages");
    const receivedData = JSON.parse(event.data);
    console.log('Получено сообщение от сервера:', receivedData);
    console.log('Тип сообщения от сервера:', receivedData.type);
    if (receivedData.message) {
        if (receivedData.message === "Неверный логин или пароль") {
            alert(receivedData.message);
            isAuthorized = false;    
        }else if( receivedData.message === "Пользователь с таким именем уже существует"){
            alert(receivedData.message);
            isAuthorized = false;
        } 
        else if (receivedData.message === "Авторизация успешна" || receivedData.message === "Регистрация успешна") {
            isAuthorized = true;
            entry.classList.add('hidden');
            chatSection.classList.remove('hidden');
        } else if(receivedData.message === "Пользователь уже подключен"){
            alert("Пользователь уже подключен");
            isAuthorized = false;
        } else {
            console.log("неизвестное сообщение" + receivedData.message);
        }
    }

    if (isAuthorized) {
        const messageContainer = document.createElement("div");
        if (receivedData.name === username) {
            messageContainer.classList.add("message-container-sent"); 
        } else if ((receivedData.name === undefined && receivedData.type != "server")||(receivedData.name===""&&receivedData.type==="Server")||(receivedData.name===null&&receivedData.type==="Server")) {
            messageContainer.classList.add("message-container-server"); 
        } else if(receivedData.name != username && receivedData.type != "server") {
            messageContainer.classList.add("message-container-received"); 
        }
        
        const clientName = document.createElement("div");
        if (receivedData.name === username) {
            clientName.textContent = ` ${username}`; 
        }
         else if(receivedData.name!=undefined) {
            clientName.textContent = ` ${receivedData.name}`;
        }

     
        if (receivedData.type != "server") {
            const messageText = document.createElement("div");
            messageText.className = "message-text";
            messageText.textContent = receivedData.message;
            messageContainer.appendChild(clientName);
            messageContainer.appendChild(messageText);

           
            messagesDiv.appendChild(messageContainer);

            messagesDiv.scrollTop = messagesDiv.scrollHeight;
        }
    }
}




send_button.addEventListener("click", sendMessage);
let password = ""
function sendMessage(evt) {
    if (input.value.trim() !== "") {
        const messageObject = {
            name: username,
            message: input.value.trim()
        };

        const jsonMessage = JSON.stringify(messageObject);

        socket.send(jsonMessage);
        input.value = ""; 

        const messagesDiv = document.getElementById("messages");
        messagesDiv.scrollTop = messagesDiv.scrollHeight;
    }
}



input.addEventListener('keydown', keyHandler);

function keyHandler(evt) {
    if (evt.key === 'Enter') {
        sendMessage(evt);
    }
}
