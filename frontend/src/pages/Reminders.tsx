import { useNavigate } from "react-router-dom";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { api } from "@/lib/api";
import { fmtDate, fmtDateTime } from "@/lib/format";
import type { ReminderLog } from "@/lib/types";
import { usePageTitle } from "@/lib/hooks";

export default function Reminders() {
  usePageTitle("提醒记录");
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { data: reminders, isLoading, error } = useQuery({
    queryKey: ["reminders"],
    queryFn: () => api<ReminderLog[]>("/api/reminders"),
  });

  const dismissMutation = useMutation({
    mutationFn: (id: number) =>
      api<ReminderLog>(`/api/reminders/${id}/dismiss`, { method: "POST" }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["reminders"] }),
  });

  if (isLoading) return <p className="text-muted-foreground">加载中…</p>;
  if (error) return <p className="text-destructive">{error instanceof Error ? error.message : "加载失败"}</p>;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">提醒记录</h1>
        <p className="text-sm text-muted-foreground">
          到期前 30 / 7 / 1 天各发送一封邮件，每档只发一次
        </p>
      </div>

      <div className="rounded-md border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>资产</TableHead>
              <TableHead>到期日期</TableHead>
              <TableHead>提前天数</TableHead>
              <TableHead>发送时间</TableHead>
              <TableHead>状态</TableHead>
              <TableHead className="w-24" />
            </TableRow>
          </TableHeader>
          <TableBody>
            {(reminders ?? []).length === 0 && (
              <TableRow>
                <TableCell colSpan={6} className="py-10 text-center text-muted-foreground">
                  暂无提醒记录，配置 SMTP 后到期前会自动发送邮件
                </TableCell>
              </TableRow>
            )}
            {(reminders ?? []).map((reminder) => (
              <TableRow key={reminder.id}>
                <TableCell>
                  <button
                    className="font-medium hover:underline"
                    onClick={() => navigate(`/assets/${reminder.asset_id}`)}
                  >
                    {reminder.asset_name}
                  </button>
                </TableCell>
                <TableCell>{fmtDate(reminder.target_date)}</TableCell>
                <TableCell>{reminder.lead_days} 天前</TableCell>
                <TableCell className="text-muted-foreground">{fmtDateTime(reminder.sent_at)}</TableCell>
                <TableCell>
                  {!reminder.sent ? (
                    <Badge variant="destructive">发送失败，次日重试</Badge>
                  ) : reminder.dismissed ? (
                    <Badge variant="secondary">已忽略</Badge>
                  ) : (
                    <Badge variant="default">已发送</Badge>
                  )}
                </TableCell>
                <TableCell>
                  {!reminder.dismissed && (
                    <Button
                      variant="ghost"
                      size="sm"
                      disabled={dismissMutation.isPending}
                      onClick={() => dismissMutation.mutate(reminder.id)}
                    >
                      忽略
                    </Button>
                  )}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>
    </div>
  );
}
