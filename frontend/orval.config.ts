import { defineConfig } from "orval";

export default defineConfig({
  auth: {
    input: "../api/auth.openapi.yaml",
    output: {
      target: "./src/services/auth/api.gen.ts",
      mode: "single",
      schemas: "./src/model/api",
      client: "axios",
      // fileExtension: ".gen.ts",
      namingConvention: "kebab-case",
      override: {
        mutator: {
          path: "./src/lib/client.ts",
          name: "getClient",
        },
      },
    },
  },
});
