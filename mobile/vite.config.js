import { defineConfig } from "vite";

export default defineConfig({
  base: "/wallet/",
  server: {
    port: 4174,
    strictPort: false,
  },
  preview: {
    port: 4174,
    strictPort: false,
  },
});
