import ws from 'k6/ws';
import { check } from 'k6';

export const options = {
    vus: 100, // количество виртуальных пользователей
    duration: '1m', // время выполнения теста
};

export default function () {
    const url = 'ws://localhost:8080/ws'; 
    const params = { tags: { name: 'WebSocketTest' } };

    const response = ws.connect(url, params, function (socket) {
        socket.on('open', function () {
            console.log('Соединение установлено');


            const registrationData = JSON.stringify({
                name: `TestUser_${__VU}_${Date.now()}`, 
                password: 'password123', 
                is_register: true, 
            });

            socket.send(registrationData);
            console.log('Данные для регистрации отправлены:', registrationData);
        });

        socket.on('message', function (message) {
            console.log('Ответ от сервера:', message);

         
            const serverResponse = JSON.parse(message);
            check(serverResponse, {
                'Ответ содержит сообщение': (resp) => resp.message !== undefined,
                'Регистрация успешна': (resp) => resp.message === 'Регистрация успешна',
            });

            socket.close();
        });

        socket.on('close', function () {
            console.log('Соединение закрыто');
        });

        socket.on('error', function (e) {
            console.error('Ошибка:', e.error());
        });
    });

    check(response, { 'Соединение установлено': (res) => res && res.status === 101 });
}
