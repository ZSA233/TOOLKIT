import { mountWailsApp } from "./app/wails_mount";
import { DashboardApp } from "./features/dashboard/DashboardApp";
import { createDashboardDeps } from "./features/dashboard/deps";
import "@toolkit/appkit-webui/draft-list.css";
import "./styles.css";

mountWailsApp({
  app: <DashboardApp deps={createDashboardDeps()} />,
});
