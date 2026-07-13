// @ts-ignore
import { createRoot } from "react-dom/client";
import { BrowserRouter } from "react-router-dom";
import "./index.css";
import Root from "~/pages/Root";

const root = createRoot(document.querySelector("#root")!);
root.render(<Root Router={BrowserRouter} />);
