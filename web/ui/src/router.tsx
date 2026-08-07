import { createBrowserRouter, Navigate, Outlet } from "react-router-dom"
import { NuqsAdapter } from "nuqs/adapters/react-router/v7"
import { Dashboard } from "./routes/dashboard"
import { SourcesLayout } from "./routes/sources-layout"
import { DeliveriesLayout } from "./routes/deliveries-layout"
import { EmptyState } from "./routes/empty-state"
import { RouteError } from "./routes/error-boundary"
import { SourceDetail } from "./routes/source-detail"
import { SourceOverview } from "./routes/source-overview"
import { SourceActions } from "./routes/source-actions"
import { SourceScript } from "./routes/source-script"
import { SourceEvents } from "./routes/source-events"
import { DeliveryDetail } from "./routes/delivery-detail"

function RootLayout() {
  return (
    <NuqsAdapter>
      <Outlet />
    </NuqsAdapter>
  )
}

export const router = createBrowserRouter([
  {
    element: <RootLayout />,
    children: [
      { path: "/", element: <Dashboard />, errorElement: <RouteError /> },
      {
        path: "/sources", element: <SourcesLayout />, errorElement: <RouteError />,
        children: [
          { index: true, element: <EmptyState title="Select a source" /> },
          {
            path: ":slug", element: <SourceDetail />,
            children: [
              { index: true, element: <Navigate to="overview" replace /> },
              { path: "overview", element: <SourceOverview /> },
              { path: "actions", element: <SourceActions /> },
              { path: "script", element: <SourceScript /> },
              { path: "events", element: <SourceEvents /> },
            ],
          },
        ],
      },
      {
        path: "/deliveries", element: <DeliveriesLayout />, errorElement: <RouteError />,
        children: [
          { index: true, element: <EmptyState title="Select a delivery" /> },
          { path: ":id", element: <DeliveryDetail /> },
        ],
      },
    ],
  },
])
