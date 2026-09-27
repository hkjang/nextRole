import React from "react";
import ReactDOM from "react-dom/client";
import { MantineProvider, createTheme } from "@mantine/core";
import { BrowserRouter } from "react-router-dom";
import "@mantine/core/styles.css";
import "@fontsource/noto-sans-kr/korean-400.css";
import "@fontsource/noto-sans-kr/korean-500.css";
import "@fontsource/noto-sans-kr/korean-600.css";
import "@fontsource/noto-sans-kr/korean-700.css";
import "./styles.css";
import App from "./App";
const theme = createTheme({
  fontFamily: '"Noto Sans KR", sans-serif',
  primaryColor: "teal",
  primaryShade: 9,
  defaultRadius: "md",
  components: {
    Modal: { defaultProps: { closeButtonProps: { "aria-label": "닫기" } } },
  },
  fontSizes: {
    xs: "0.8125rem",
    sm: "0.9375rem",
    md: "1rem",
    lg: "1.125rem",
    xl: "1.25rem",
  },
  headings: { fontFamily: '"Noto Sans KR", sans-serif' },
});
ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <MantineProvider theme={theme}>
      <BrowserRouter>
        <App />
      </BrowserRouter>
    </MantineProvider>
  </React.StrictMode>,
);
