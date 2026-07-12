import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig({
  plugins: [react()],
  build: {
    outDir: "dist",
    emptyOutDir: true,
    rollupOptions: {
      output: {
        manualChunks: {
          icons: ["lucide-react"],
          markdown: ["react-markdown", "remark-gfm"],
          react: ["react", "react-dom"],
        },
      },
    },
  },
});
