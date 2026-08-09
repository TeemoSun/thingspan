import { NavLink, Outlet, useNavigate } from "react-router-dom";
import { IconBell, IconDevices, IconLayoutDashboard, IconLogout, IconTags } from "@tabler/icons-react";

import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { logout } from "@/lib/api";

const NAV_ITEMS = [
  { to: "/", label: "仪表盘", icon: IconLayoutDashboard, end: true },
  { to: "/assets", label: "资产", icon: IconDevices },
  { to: "/categories", label: "类别", icon: IconTags },
  { to: "/reminders", label: "提醒记录", icon: IconBell },
];

export default function AppShell() {
  const navigate = useNavigate();
  return (
    <div className="flex min-h-screen">
      <aside className="fixed inset-y-0 left-0 z-30 flex w-52 flex-col border-r bg-background">
        <div className="flex h-14 items-center gap-2 border-b px-4">
          <IconDevices className="h-5 w-5" />
          <span className="text-base font-semibold">Thingspan</span>
        </div>
        <nav className="flex-1 space-y-1 p-3">
          {NAV_ITEMS.map((item) => (
            <NavLink
              key={item.to}
              to={item.to}
              end={item.end}
              className={({ isActive }) =>
                cn(
                  "flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors",
                  isActive ? "bg-accent text-accent-foreground" : "text-muted-foreground hover:bg-accent/50 hover:text-foreground"
                )
              }
            >
              <item.icon className="h-4 w-4" />
              {item.label}
            </NavLink>
          ))}
        </nav>
        <div className="border-t p-3">
          <Button
            variant="ghost"
            className="w-full justify-start text-muted-foreground"
            onClick={() => {
              logout();
              navigate("/login");
            }}
          >
            <IconLogout className="h-4 w-4" />
            退出登录
          </Button>
        </div>
      </aside>
      <main className="ml-52 flex-1 p-6">
        <Outlet />
      </main>
    </div>
  );
}
