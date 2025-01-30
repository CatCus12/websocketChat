import { check, sleep } from 'k6';
import ws from 'k6/ws';
import { SharedArray } from 'k6/data';


const messages = new SharedArray('Messages', function () {
  return JSON.parse(open('./messages.json')); 
});


export const options = {
  scenarios: {
    websocket_test: {
      executor: 'constant-vus', 
      vus: 10,               
      duration: '30s',         
    },
  },
};

export default function () {
  const url = 'ws://localhost:8080/ws';
  const userIndex = __VU % messages.length; 
  const messageData = messages[userIndex]; 

  const res = ws.connect(url, {}, function (socket) {
    socket.on('open', function () {
      console.log(`Connection opened for user: ${messageData.name}`);
      socket.send(JSON.stringify(messageData)); 
    });

    socket.on('message', function (msg) {
      console.log(`Received message: ${msg}`);
    });

    socket.on('close', function () {
      console.log(`Connection closed for user: ${messageData.name}`);
    });

    socket.on('error', function (e) {
      console.error(`WebSocket error for user ${messageData.name}:`, e);
    });

    sleep(1); 
  });

  check(res, { 'connection established': (r) => r && r.status === 101 });
  sleep(1); 
}
