import { defineConfig } from "orval";

export default defineConfig({
  api: {
    input: "../../spec/generated/openapi.yaml",
    output: {
      target: "./src/api/generated/client.ts",
      client: "react-query",
      httpClient: "fetch",
      clean: true,
      prettier: false,
      override: {
        query: {
          signal: true,
        },
      },
    },
  },
});
