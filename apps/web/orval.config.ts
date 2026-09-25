import { defineConfig } from "orval";

export default defineConfig({
  enterprise: {
    input: "../../spec/generated/openapi.yaml",
    output: {
      target: "./src/api/generated/enterprise.ts",
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
