import { createBrowserRouter, Navigate } from "react-router-dom"
import { SourcesLayout } from "./routes/sources-layout"
import { DeliveriesLayout } from "./routes/deliveries-layout"
import { EmptyState } from "./routes/empty-state"
import { SourceDetail } from "./routes/source-detail"
import { SourceOverview } from "./routes/source-overview"
import { SourceActions } from "./routes/source-actions"
import { SourceScript } from "./routes/source-script"
import { SourceEvents } from "./routes/source-events"
import { DeliveryDetail } from "./routes/delivery-detail"

export const router = createBrowserRouter([
  { path: "/", element: <Navigate to="/sources" replace /> },
  {
    path: "/sources", element: <SourcesLayout />,
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
    path: "/deliveries", element: <DeliveriesLayout />,
    children: [
      { index: true, element: <EmptyState title="Select a delivery" /> },
      { path: ":id", element: <DeliveryDetail /> },
    ],
  },
])
