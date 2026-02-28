import index from "./index.html";

Bun.serve({
  port: 5173,
  routes: {
    "/*": index,
  },
  development: {
    hmr: true,
    console: true,
  },
});

console.log("Aule dashboard running on http://localhost:5173");
