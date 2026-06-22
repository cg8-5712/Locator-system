import { Navigate, createBrowserRouter } from "react-router-dom";
import { RootLayout } from "../components/shell/root-layout";
import { ProtectedLayout } from "../components/shell/protected-layout";
import { LoginPage } from "../pages/login/login-page";
import { MapPage } from "../pages/map/map-page";

export const appRouter = createBrowserRouter([
  {
    path: "/",
    element: <RootLayout />,
    children: [
      { index: true, element: <Navigate to="/app/map" replace /> },
      { path: "login", element: <LoginPage /> },
      {
        path: "app",
        element: <ProtectedLayout />,
        children: [{ path: "map", element: <MapPage /> }],
      },
    ],
  },
]);
