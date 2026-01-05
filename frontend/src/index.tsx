import { serve } from "bun";
import index from "./index.html";

interface WebSocketData {}

const API_ADDR = "localhost:9000";

const server = serve({
  routes: {
    "/*": index,

    "/api/ws": async (req, server) => {
      if (server.upgrade(req, { data: {} })) {
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
    open: (socket) => {
      console.log("WebSocket connection established");
      socket.proxy = new WebSocket(`ws://${API_ADDR}/api/ws`);
    },
    message: (socket, message) => {
      console.log("Received message:", message);
      socket.send(`Server received: ${message}`);
    },
    close: (socket, code, reason) => {
      console.log(`WebSocket closed: ${code} - ${reason}`);
    },
  },

  development: process.env.NODE_ENV !== "production" && {
    hmr: true,
    console: true,
  },
});

console.log(`🚀 Server running at ${server.url}`);
