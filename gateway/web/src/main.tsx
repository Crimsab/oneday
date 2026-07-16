import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import "./i18n";
import "@fontsource-variable/ibm-plex-sans";
import App from "./App";
import "./styles.css";
import "./redesign.css";
import { installSupportDiagnostics } from "./supportDiagnostics";
import { PwaStatus } from "./features/pwa/PwaStatus";

installSupportDiagnostics();

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <App />
    <PwaStatus />
  </StrictMode>,
);
