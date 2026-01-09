import { serve } from "bun";
import index from "./index.html";

interface WebSocketData {
  url: URL;
}

type ServerWebSocket = Bun.ServerWebSocket<WebSocketData> & {
  proxy: WebSocket;
};

const API_ADDR = "localhost:9000";

const server = serve({
  routes: {
    "/*": index,

    "/api/ws": async (req, server) => {
      const url = new URL(req.url);
      if (
        server.upgrade(req, {
          data: {
            url,
          },
        })
      ) {
        return;
      }

      return new Response("Connection must upgrade", { status: 400 });
    },

    "/api/*": async (req) => {
      const url = new URL(req.url);
      const targetUrl = new URL(
        url.pathname + url.search,
        `http://${API_ADDR}`,
      );

      return fetch(targetUrl, {
        method: req.method,
        headers: req.headers,
        body: req.method === "GET" || req.method === "HEAD" ? null : req.body,
      });
    },
  },

  websocket: {
    data: {} as WebSocketData,
    maxPayloadLength: 16 * 1024 * 1024,
    sendPings: false,
    open: (socket: ServerWebSocket) => {
      const url = socket.data.url;
      const targetUrl = new URL(url.pathname + url.search, `ws://${API_ADDR}`);

      console.log("WebSocket connection established");
      socket.proxy = new WebSocket(targetUrl.href);
      socket.proxy.onopen = () => {
        console.log("Connected to backend WebSocket");
      };
      socket.proxy.onmessage = (event) => {
        socket.send(event.data);
      };
      socket.proxy.onclose = (event) => {
        console.log(
          `Backend WebSocket closed: ${event.code} - ${event.reason}`,
        );
        socket.close(event.code, event.reason);
      };
    },
    message: (socket: ServerWebSocket, message) => {
      console.log("Received message:", message);
      socket.proxy.send(message);
    },
    close: (socket: ServerWebSocket, code, reason) => {
      console.log(`WebSocket closed: ${code} - ${reason}`);
      socket.proxy.close();
    },
  },

  development: process.env.NODE_ENV !== "production" && {
    hmr: true,
    console: true,
  },
});

console.log(`🚀 Server running at ${server.url}`);
