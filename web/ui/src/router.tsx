import { createBrowserRouter, Navigate } from "react-router-dom"
import { SourcesLayout } from "./routes/sources-layout"
import { DeliveriesLayout } from "./routes/deliveries-layout"
import { EmptyState } from "./routes/empty-state"

export const router = createBrowserRouter([
  { path: "/", element: <Navigate to="/sources" replace /> },
  {
    path: "/sources", element: <SourcesLayout />,
    children: [{ index: true, element: <EmptyState title="Select a source" /> }],
  },
  {
    path: "/deliveries", element: <DeliveriesLayout />,
    children: [{ index: true, element: <EmptyState title="Select a delivery" /> }],
  },
])
