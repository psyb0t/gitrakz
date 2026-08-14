import { mount } from "svelte";
import "./app.css";
import App from "./App.svelte";

const target = document.getElementById("app");
if (!target) {
  // No empty catches/silent nulls per the error-handling mandate — a
  // missing mount point means index.html and this bundle drifted apart,
  // which is unrecoverable, so fail loud instead of a blank white page.
  throw new Error('mount target "#app" not found in index.html');
}

const app = mount(App, { target });

export default app;
