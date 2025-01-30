import { check, sleep } from 'k6';
import ws from 'k6/ws';

export default function () {
  const url = 'ws://localhost:8080/ws';

  const res = ws.connect(url, {}, function (socket) {
    socket.on('open', function () {
      console.log('WebSocket connection opened');

    
      const authMessage = JSON.stringify({
        name: "Tea",
        password: "thr",
        is_register: false
      });
      console.log('Sending auth message:', authMessage);
      socket.send(authMessage);
    });

   
    socket.on('message', function (response) {
      console.log('Server response:', response); 
      
     
      const isSuccess = check(response, {
        'Авторизация успешна': (r) => {
          try {
            const jsonResponse = JSON.parse(r);
            console.log('Parsed server response:', jsonResponse); 
            return jsonResponse.message === "Авторизация успешна";
          } catch (error) {
            console.error('Failed to parse server response:', error);
            return false;
          }
        },
      });

      console.log('Authorization success check result:', isSuccess);

      
      socket.close();
    });

    socket.on('error', function (e) {
      console.error('WebSocket error:', e); 
    });

    socket.on('close', function () {
      console.log('WebSocket connection closed');
    });
  });

  check(res, { 'connection established': (r) => r && r.status === 101 });
  sleep(1);
}
