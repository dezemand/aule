import index from "./index.html";

const port = Number(process.env.PORT) || 5173;

Bun.serve({
  port,
  routes: {
    "/*": index,
  },
  development: {
    hmr: true,
    console: true,
  },
});

console.log(`Aule dashboard running on http://localhost:${port}`);
