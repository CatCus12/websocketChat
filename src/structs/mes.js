import { check, sleep } from 'k6';
import ws from 'k6/ws';

export default function () {
  const url = 'ws://localhost:8080/ws'; 

  const res = ws.connect(url, {}, function (socket) {
    let isAuthorized = false;

    socket.on('open', function () {
      console.log('WebSocket connection opened');

     
      const authMessage = JSON.stringify({
        name: 'Tea',
        password: 'thr',
        is_register: false,
      });
      console.log('Sending auth message:', authMessage);
      socket.send(authMessage);
    });

   
    socket.on('message', function (response) {
      console.log('Server response:', response);

      try {
        const jsonResponse = JSON.parse(response);

        
        if (!isAuthorized && jsonResponse.message === 'Авторизация успешна') {
          console.log('Authorization successful!');
          isAuthorized = true;

          
          const chatMessage = JSON.stringify({
            name: 'Tea',
            message: 'Hello, chat!',
          });
          console.log('Sending chat message:', chatMessage);
          socket.send(chatMessage);

        } else if (isAuthorized) {
          
          const isMessageReceived = check(jsonResponse, {
            'Сообщение успешно обработано сервером': (r) =>
              r.message === 'Hello, chat!',
          });
          console.log('Message processing success check:', isMessageReceived);

          
          setTimeout(() => socket.close(), 100);
        }
      } catch (error) {
        console.error('Error parsing server response:', error);
        socket.close();
      }
    });

    socket.on('error', function (e) {
      console.error('WebSocket error:', e);
    });

    socket.on('close', function (code, reason) {
      console.log(`WebSocket connection closed. Code: ${code}, Reason: ${reason}`);
    });
  });

  
  check(res, { 'connection established': (r) => r && r.status === 101 });

  sleep(1); 
}
