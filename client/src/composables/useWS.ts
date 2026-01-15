import Sockette from "sockette";
import { getWebSocketUrl } from '../config/websocket';

let ws: Sockette | null = null;
const subscriptions: Record<string, Array<Function>> = {};

export function useWS() {
 
  function init() {
    // If already connected, don't create a new connection
    if (ws) {
      return;
    }
    
    ws = new Sockette(getWebSocketUrl(), {
      timeout: 5e3,
      maxAttempts: 10,
      onopen: e => {},
      onmessage: e => {
        if(e.data) {
          const message = JSON.parse(e.data);
          if(subscriptions[message.type] && subscriptions[message.type].length > 0) {
            for(let cb of subscriptions[message.type]) {
              cb(message);
            }
          }
        }
      },
      onreconnect: e => {},
      onmaximum: e => {},
      onclose: e => {
        ws = null; // Reset so next init() creates a new connection
      },
      onerror: e => {}
    });
  }

  function send(type: string, content?: any) {
    if(ws) {
      ws.json({ type, content });
    }
  }

  function on(eventName: string, callback: Function) {
    if (!subscriptions[eventName]) {
      subscriptions[eventName] = [];
    }
    subscriptions[eventName].push(callback);
  }

  function off(eventName: string, callback?: Function) {
    if (!subscriptions[eventName]) return;
    
    if (callback) {
      // Remove specific callback
      subscriptions[eventName] = subscriptions[eventName].filter(cb => cb !== callback);
    } else {
      // Remove all callbacks for this event
      delete subscriptions[eventName];
    }
  }

  function close() {
    if(ws) {
      ws.close();
      ws = null;
    }
    // Clear all subscriptions
    Object.keys(subscriptions).forEach(key => {
      delete subscriptions[key];
    });
  }

  function isConnected(): boolean {
    return ws !== null;
  }
  
  return {
    init,
    send,
    close,
    on,
    off,
    isConnected,
  }
}