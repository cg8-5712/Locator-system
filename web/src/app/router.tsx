import type { ReactNode } from "react";
import { Navigate, createBrowserRouter } from "react-router-dom";
import { RootLayout } from "../components/shell/root-layout";
import { ProtectedLayout } from "../components/shell/protected-layout";
import type { MapDataSource } from "../features/map-view/data-source";
import { MapDataProvider } from "../features/map-view/map-data-context";
import { demoDataSource } from "../features/map-view/demo-data-source";
import { liveDataSource } from "../features/map-view/live-data-source";
import { LoginPage } from "../pages/login/login-page";
import { MapPage } from "../pages/map/map-page";
import { AlarmsPage } from "../pages/alarms/alarms-page";
import { HistoryPage } from "../pages/history/history-page";
import { DemoSharePage } from "../pages/share/demo-share-page";

function withDataSource(dataSource: MapDataSource, element: ReactNode) {
  return <MapDataProvider value={dataSource}>{element}</MapDataProvider>;
}

export const appRouter = createBrowserRouter([
  {
    path: "/",
    element: <RootLayout />,
    children: [
      { index: true, element: <Navigate to="/demo/map" replace /> },
      { path: "login", element: <LoginPage /> },
      { path: "demo", element: <Navigate to="/demo/map" replace /> },
      {
        path: "demo/map",
        element: withDataSource(demoDataSource, <MapPage />),
      },
      {
        path: "demo/alarms",
        element: withDataSource(demoDataSource, <AlarmsPage />),
      },
      {
        path: "demo/devices/:deviceSN/history",
        element: withDataSource(demoDataSource, <HistoryPage />),
      },
      {
        path: "demo/share/:deviceSN",
        element: <DemoSharePage />,
      },
      {
        path: "app",
        element: <ProtectedLayout />,
        children: [
          { index: true, element: <Navigate to="/app/map" replace /> },
          {
            path: "map",
            element: withDataSource(liveDataSource, <MapPage />),
          },
          {
            path: "alarms",
            element: withDataSource(liveDataSource, <AlarmsPage />),
          },
          {
            path: "devices/:deviceSN/history",
            element: withDataSource(liveDataSource, <HistoryPage />),
          },
        ],
      },
    ],
  },
]);
