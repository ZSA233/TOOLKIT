import { screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";

import { mountWailsApp } from "./app/wails_mount";

afterEach(() => {
  document.body.innerHTML = "";
});

describe("mountWailsApp", () => {
  it("renders the provided app into the default root element", async () => {
    document.body.innerHTML = '<div id="root"></div>';

    const root = mountWailsApp({
      app: <main>shared shell</main>,
    });

    await waitFor(() => {
      expect(screen.getByText("shared shell")).toBeTruthy();
    });

    root.unmount();
  });

  it("throws a clear error when the root element is missing", () => {
    expect(() =>
      mountWailsApp({
        app: <main>missing root</main>,
      }),
    ).toThrowError("Unable to mount Wails tool app: missing root element '#root'.");
  });
});
