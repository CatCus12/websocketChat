import { check } from 'k6';
import ws from 'k6/ws';
import { sleep } from 'k6';

// Сценарий с фиксированным количеством виртуальных пользователей
export const options = {
  scenarios: {
    websocket_test: {
      executor: 'constant-vus',  // Тип сценария - фиксированное количество виртуальных пользователей
      vus: 100,                    // Количество виртуальных пользователей
      duration: '1m',           
    },
  },
};

export default function () {
  const url = 'ws://localhost:8080/ws';

  // Сначала Tea, затем Tea1, для каждого пользователя по очереди
  const userIndex = __VU - 1;  // Уменьшаем на 1, чтобы индекс начинался с 0
  const user = userIndex % 2 === 0 ? { name: 'Tea', password: 'thr' } : { name: 'Tea1', password: 'thr' };

  const res = ws.connect(url, {}, function (socket) {
    
    socket.on('open', function () {
      // Отправка сообщения для авторизации
      socket.send(
        JSON.stringify({
          name: user.name,
          password: user.password,
          is_register: false,
        })
      );
    });

    // Обработка сообщений от сервера
    socket.on('message', function (response) {
      const parsedResponse = JSON.parse(response);
      check(parsedResponse, {
        
      });
      sleep(2); // Задержка перед закрытием соединения
      socket.close();
    });

    socket.on('error', function (e) {
      console.log('WebSocket error:', e);
    });

    socket.on('close', function () {
      console.log(`Connection closed for user: ${user.name}`);
    });
  });

  check(res, { 'connection established': (r) => r && r.status === 101 });
  sleep(10); // Задержка между итерациями
}
