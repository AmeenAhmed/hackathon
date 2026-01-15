import Sockette from "sockette";

let ws: Sockette | null = null;

export function useWS() {
 
  function init() {
    ws = new Sockette('ws://localhost:8080/ws', {
      timeout: 5e3,
      maxAttempts: 10,
      onopen: e => console.log('Connected!', e),
      onmessage: e => console.log('Received:', e),
      onreconnect: e => console.log('Reconnecting...', e),
      onmaximum: e => console.log('Stop Attempting!', e),
      onclose: e => console.log('Closed!', e),
      onerror: e => console.log('Error:', e)
    });
  }

  function send(type: string, data: any) {
    if(ws) {
      ws.json({ type, data });
    }
  }

  function close() {
    if(ws) {
      ws.close();
    }
  }
  
  return {
    init,
    send,
    close
  }
}