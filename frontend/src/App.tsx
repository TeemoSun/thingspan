import type { ReactElement } from "react";
import { Navigate, Route, Routes } from "react-router-dom";

import AppShell from "@/components/AppShell";
import { isLoggedIn } from "@/lib/api";
import AssetDetail from "@/pages/AssetDetail";
import Assets from "@/pages/Assets";
import Categories from "@/pages/Categories";
import Dashboard from "@/pages/Dashboard";
import Login from "@/pages/Login";
import Reminders from "@/pages/Reminders";

function RequireAuth({ children }: { children: ReactElement }) {
  return isLoggedIn() ? children : <Navigate to="/login" replace />;
}

export default function App() {
  return (
    <Routes>
      <Route path="/login" element={<Login />} />
      <Route
        element={
          <RequireAuth>
            <AppShell />
          </RequireAuth>
        }
      >
        <Route index element={<Dashboard />} />
        <Route path="/assets" element={<Assets />} />
        <Route path="/assets/new" element={<AssetDetail />} />
        <Route path="/assets/:id" element={<AssetDetail />} />
        <Route path="/categories" element={<Categories />} />
        <Route path="/reminders" element={<Reminders />} />
      </Route>
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
}
