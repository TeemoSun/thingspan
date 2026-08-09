import { Link } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import {
  IconBell,
  IconCalendarStats,
  IconCoin,
  IconDevices,

} from "@tabler/icons-react";
import dayjs from "dayjs";

import { Badge } from "@/components/ui/badge";

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { api } from "@/lib/api";
import { fmtMoney, fmtMoneyShort } from "@/lib/format";
import type { Dashboard } from "@/lib/types";
import { usePageTitle } from "@/lib/hooks";

function StatCard({
  title,
  value,
  sub,
  icon: Icon,
}: {
  title: string;
  value: string;
  sub?: string;
  icon: React.ComponentType<{ className?: string }>;
}) {
  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
        <CardTitle className="text-sm font-medium text-muted-foreground">{title}</CardTitle>
        <Icon className="h-4 w-4 text-muted-foreground" />
      </CardHeader>
      <CardContent>
        <div className="text-2xl font-bold">{value}</div>
        {sub && <p className="mt-1 text-xs text-muted-foreground">{sub}</p>}
      </CardContent>
    </Card>
  );
}

export default function Dashboard() {
  usePageTitle("仪表盘");
  const { data, isLoading, error } = useQuery({
    queryKey: ["dashboard"],
    queryFn: () => api<Dashboard>("/api/dashboard"),
  });

  if (isLoading) return <p className="text-muted-foreground">加载中…</p>;
  if (error) return <p className="text-destructive">{error instanceof Error ? error.message : "加载失败"}</p>;
  if (!data) return null;

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">仪表盘</h1>
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <StatCard title="资产总数" value={String(data.total_assets)} icon={IconDevices} sub={`使用中 ${data.in_use_assets} 件`} />
        <StatCard title="总投入" value={fmtMoneyShort(data.total_invested)} icon={IconCoin} />
        <StatCard title="当前日均总成本" value={fmtMoney(data.daily_cost_total)} icon={IconCalendarStats} sub="所有在用资产合计" />
        <StatCard
          title="30 天内到期"
          value={String(data.expiring_soon.length)}
          icon={IconBell}
          sub={data.expiring_soon.length > 0 ? "即将到期，注意查收邮件" : "暂无临近到期的资产"}
        />
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">即将到期</CardTitle>
        </CardHeader>
        <CardContent>
          {data.expiring_soon.length === 0 ? (
            <p className="py-6 text-center text-sm text-muted-foreground">30 天内没有资产到期</p>
          ) : (
            <ul className="divide-y">
              {data.expiring_soon.map((item) => {
                const date = dayjs(item.target_date);
                const isUrgent = item.days_left <= 7;
                return (
                  <li key={item.id} className="flex items-center justify-between py-3">
                    <div className="min-w-0">
                      <Link to={`/assets/${item.id}`} className="font-medium hover:underline">
                        {item.name}
                      </Link>
                      <p className="truncate text-xs text-muted-foreground">
                        {item.category_name} · {item.date_type === "warranty" ? "保修" : "到期"}至 {date.format("YYYY-MM-DD")}
                      </p>
                    </div>
                    <Badge variant={isUrgent ? "destructive" : "warning"}>
                      {item.days_left === 0 ? "今天到期" : `${item.days_left} 天`}
                    </Badge>
                  </li>
                );
              })}
            </ul>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
