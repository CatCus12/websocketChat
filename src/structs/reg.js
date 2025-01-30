import { check, sleep } from 'k6';
import ws from 'k6/ws';

export default function () {
  const url = 'ws://localhost:8080/ws';

  const res = ws.connect(url, {}, function (socket) {
    socket.on('open', function () {
      console.log('WebSocket connection opened');

      // Отправка сообщения для авторизации
      const authMessage = JSON.stringify({
        name: "newUserdmsfklaslf",
        password: "thr",
        is_register: true
      });
      console.log('Sending auth message:', authMessage);
      socket.send(authMessage);
    });

    // Обработка сообщений от сервера
    socket.on('message', function (response) {
      console.log('Server response:', response); // Логируем весь ответ сервера
      
      // Проверяем ответ сервера на успешность авторизации
      const isSuccess = check(response, {
        'Регистрация успешна': (r) => {
          try {
            const jsonResponse = JSON.parse(r);
            console.log('Parsed server response:', jsonResponse); // Логируем парсинг
            return jsonResponse.message === "Регистрация успешна";
          } catch (error) {
            console.error('Failed to parse server response:', error);
            return false;
          }
        },
      });

      console.log('Registration success check result:', isSuccess);

      // Закрываем WebSocket после обработки сообщения
      socket.close();
    });

    socket.on('error', function (e) {
      console.error('WebSocket error:', e); // Логируем ошибки
    });

    socket.on('close', function () {
      console.log('WebSocket connection closed');
    });
  });

  check(res, { 'connection established': (r) => r && r.status === 101 });
  sleep(1);
}
