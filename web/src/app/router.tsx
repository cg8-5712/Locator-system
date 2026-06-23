import { Navigate, createBrowserRouter } from "react-router-dom";
import { RootLayout } from "../components/shell/root-layout";
import { ProtectedLayout } from "../components/shell/protected-layout";
import { MapDataProvider } from "../features/map-view/map-data-context";
import { demoDataSource } from "../features/map-view/demo-data-source";
import { liveDataSource } from "../features/map-view/live-data-source";
import { LoginPage } from "../pages/login/login-page";
import { MapPage } from "../pages/map/map-page";

export const appRouter = createBrowserRouter([
  {
    path: "/",
    element: <RootLayout />,
    children: [
      { index: true, element: <Navigate to="/demo/map" replace /> },
      { path: "login", element: <LoginPage /> },
      {
        path: "demo/map",
        element: (
          <MapDataProvider value={demoDataSource}>
            <MapPage />
          </MapDataProvider>
        ),
      },
      {
        path: "app",
        element: <ProtectedLayout />,
        children: [
          {
            path: "map",
            element: (
              <MapDataProvider value={liveDataSource}>
                <MapPage />
              </MapDataProvider>
            ),
          },
        ],
      },
    ],
  },
]);
