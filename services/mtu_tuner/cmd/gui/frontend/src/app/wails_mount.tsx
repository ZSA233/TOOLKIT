import { StrictMode, type ReactNode } from "react";
import { createRoot, type Root } from "react-dom/client";

type MountWailsAppOptions = {
  app: ReactNode;
  rootElement?: HTMLElement | null;
  rootElementId?: string;
};

function WailsAppRoot(props: { children: ReactNode }) {
  return <StrictMode>{props.children}</StrictMode>;
}

function resolveRootElement(
  rootElement: HTMLElement | null | undefined,
  rootElementId: string,
): HTMLElement {
  if (rootElement) {
    return rootElement;
  }

  const resolvedRootElement = document.getElementById(rootElementId);
  if (resolvedRootElement) {
    return resolvedRootElement;
  }

  // Keep the bootstrap failure mode explicit so the Wails HTML shell stays easy to diagnose.
  throw new Error(`Unable to mount Wails tool app: missing root element '#${rootElementId}'.`);
}

export function mountWailsApp({
  app,
  rootElement,
  rootElementId = "root",
}: MountWailsAppOptions): Root {
  const resolvedRootElement = resolveRootElement(rootElement, rootElementId);
  const root = createRoot(resolvedRootElement);

  root.render(<WailsAppRoot>{app}</WailsAppRoot>);

  return root;
}
