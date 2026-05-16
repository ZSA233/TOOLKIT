import path from "node:path";

import { defineConfig } from "vitest/config";
import react from "@vitejs/plugin-react-swc";

export default defineConfig({
  plugins: [react()],
  server: {
    fs: {
      allow: [path.resolve(__dirname, "../../../../../")],
    },
  },
  resolve: {
    alias: [
      {
        find: /^@toolkit\/appkit-webui\/draft-list\.css$/,
        replacement: path.resolve(
          __dirname,
          "../../../../../libs/appkit/webui/src/draft_list/draft_list.css",
        ),
      },
      {
        find: /^@toolkit\/appkit-webui$/,
        replacement: path.resolve(
          __dirname,
          "../../../../../libs/appkit/webui/src/index.ts",
        ),
      },
      {
        find: "@dnd-kit/core",
        replacement: path.resolve(__dirname, "node_modules/@dnd-kit/core"),
      },
      {
        find: "@dnd-kit/sortable",
        replacement: path.resolve(__dirname, "node_modules/@dnd-kit/sortable"),
      },
      {
        find: "@dnd-kit/utilities",
        replacement: path.resolve(__dirname, "node_modules/@dnd-kit/utilities"),
      },
      {
        find: "@testing-library/react",
        replacement: path.resolve(__dirname, "node_modules/@testing-library/react"),
      },
      {
        find: "@testing-library/user-event",
        replacement: path.resolve(__dirname, "node_modules/@testing-library/user-event"),
      },
      {
        find: "react/jsx-runtime",
        replacement: path.resolve(__dirname, "node_modules/react/jsx-runtime.js"),
      },
      {
        find: "react/jsx-dev-runtime",
        replacement: path.resolve(__dirname, "node_modules/react/jsx-dev-runtime.js"),
      },
      {
        find: "react-dom",
        replacement: path.resolve(__dirname, "node_modules/react-dom"),
      },
      {
        find: "react",
        replacement: path.resolve(__dirname, "node_modules/react"),
      },
    ],
  },
  build: {
    outDir: "dist",
    emptyOutDir: true,
  },
  test: {
    environment: "jsdom",
    testTimeout: 10000,
    include: [
      "src/**/*.test.ts",
      "src/**/*.test.tsx",
      "../../../../../libs/appkit/webui/src/**/*.test.ts",
      "../../../../../libs/appkit/webui/src/**/*.test.tsx",
    ],
  },
});
