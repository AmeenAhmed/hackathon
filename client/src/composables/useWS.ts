import Sockette from "sockette";

let ws: Sockette | null = null;
const subscriptions: Record<string, Array<Function>> = {
  'roomCreated': [],
};

export function useWS() {
 
  function init() {
    ws = new Sockette('ws://localhost:8080/ws', {
      timeout: 5e3,
      maxAttempts: 10,
      onopen: e => console.log('Connected!', e),
      onmessage: e => {
        console.log('Received:', e);
        if(e.data) {
          const message = JSON.parse(e.data);
          if(subscriptions && subscriptions[message.type] && subscriptions[message.type]?.length) {
            for(let cb of subscriptions[message.type]) {
              cb(message);
            }
          }
        }
      },
      onreconnect: e => console.log('Reconnecting...', e),
      onmaximum: e => console.log('Stop Attempting!', e),
      onclose: e => console.log('Closed!', e),
      onerror: e => console.log('Error:', e)
    });
  }

  function send(type: string, data?: any) {
    if(ws) {
      ws.json({ type, data });
    }
  }

  function on(eventName: string, callback: Function) {
    subscriptions[eventName].push(callback);
  }

  function close() {
    if(ws) {
      ws.close();
    }
  }
  
  return {
    init,
    send,
    close,
    on,
  }
}