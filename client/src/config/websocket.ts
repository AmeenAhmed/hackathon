// WebSocket configuration
export const WS_URL = import.meta.env.VITE_WS_URL || 'ws://localhost:8080/ws';

// For production builds in Docker, the WebSocket will be proxied through nginx
// In that case, we use a relative URL
export const getWebSocketUrl = () => {
  if (import.meta.env.PROD && window.location.protocol === 'http:') {
    // Production mode - use relative URL (nginx will proxy)
    return `ws://${window.location.host}/ws`;
  } else if (import.meta.env.PROD && window.location.protocol === 'https:') {
    // Production mode with HTTPS - use wss
    return `wss://${window.location.host}/ws`;
  } else {
    // Development mode - use configured URL
    return WS_URL;
  }
};