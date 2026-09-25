import { defineConfig } from "orval";

export default defineConfig({
  api: {
    input: "../../spec/generated/openapi.yaml",
    output: {
      target: "./src/api/generated/client.ts",
      client: "react-query",
      clean: true,
      prettier: false,
      override: {
        mutator: {
          path: "./src/api/client.ts",
          name: "customClient",
        },
        query: {
          signal: true,
        },
      },
    },
  },
});
