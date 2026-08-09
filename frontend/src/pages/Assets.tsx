import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { IconPlus, IconSearch } from "@tabler/icons-react";

import { StatusBadge } from "@/components/StatusBadge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { api } from "@/lib/api";
import { fmtDate, fmtMoney, fmtMoneyShort } from "@/lib/format";
import { STATUS_LABELS, type AssetList, type AssetStatus, type Category } from "@/lib/types";

export default function Assets() {
  const navigate = useNavigate();
  const [search, setSearch] = useState("");
  const [debounced, setDebounced] = useState("");
  const [categoryId, setCategoryId] = useState("");
  const [status, setStatus] = useState("");

  useEffect(() => {
    const timer = setTimeout(() => setDebounced(search), 300);
    return () => clearTimeout(timer);
  }, [search]);

  const params = new URLSearchParams();
  if (debounced) params.set("search", debounced);
  if (categoryId) params.set("category_id", categoryId);
  if (status) params.set("status", status);
  const query = params.toString();

  const { data, isLoading, error } = useQuery({
    queryKey: ["assets", query],
    queryFn: () => api<AssetList>(`/api/assets?${query}`),
  });

  const { data: categories } = useQuery({
    queryKey: ["categories"],
    queryFn: () => api<Category[]>("/api/categories"),
  });

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">资产</h1>
        <Button onClick={() => navigate("/assets/new")}>
          <IconPlus className="h-4 w-4" />
          新建资产
        </Button>
      </div>

      <div className="flex flex-wrap items-center gap-3">
        <div className="relative w-64">
          <IconSearch className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
          <Input
            placeholder="搜索名称 / 品牌 / 型号 / 序列号"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="pl-8"
          />
        </div>
        <Select value={categoryId} onValueChange={setCategoryId}>
          <SelectTrigger className="w-40">
            <SelectValue placeholder="全部类别" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="">全部类别</SelectItem>
            {(categories ?? []).map((c) => (
              <SelectItem key={c.id} value={String(c.id)}>
                {c.name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Select value={status} onValueChange={setStatus}>
          <SelectTrigger className="w-32">
            <SelectValue placeholder="全部状态" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="">全部状态</SelectItem>
            {(Object.keys(STATUS_LABELS) as AssetStatus[]).map((s) => (
              <SelectItem key={s} value={s}>
                {STATUS_LABELS[s]}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      {isLoading && <p className="text-muted-foreground">加载中…</p>}
      {error && <p className="text-destructive">{error instanceof Error ? error.message : "加载失败"}</p>}

      {data && (
        <div className="rounded-md border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>名称</TableHead>
                <TableHead>类别</TableHead>
                <TableHead>状态</TableHead>
                <TableHead>购买日期</TableHead>
                <TableHead>价格</TableHead>
                <TableHead className="text-right">日均成本</TableHead>
                <TableHead className="w-16" />
              </TableRow>
            </TableHeader>
            <TableBody>
              {data.items.length === 0 && (
                <TableRow>
                  <TableCell colSpan={7} className="py-10 text-center text-muted-foreground">
                    暂无资产，点击右上角「新建资产」开始记录
                  </TableCell>
                </TableRow>
              )}
              {data.items.map((asset) => (
                <TableRow key={asset.id}>
                  <TableCell>
                    <div className="font-medium">{asset.name}</div>
                    {asset.brand || asset.model ? (
                      <div className="text-xs text-muted-foreground">
                        {[asset.brand, asset.model].filter(Boolean).join(" · ")}
                      </div>
                    ) : null}
                  </TableCell>
                  <TableCell>{asset.category_name}</TableCell>
                  <TableCell>
                    <StatusBadge status={asset.status} />
                  </TableCell>
                  <TableCell>{fmtDate(asset.purchase_date)}</TableCell>
                  <TableCell>{fmtMoneyShort(asset.purchase_price)}</TableCell>
                  <TableCell className="text-right">{fmtMoney(asset.cost.daily_cost)}</TableCell>
                  <TableCell>
                    <Button variant="ghost" size="sm" onClick={() => navigate(`/assets/${asset.id}`)}>
                      详情
                    </Button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}
    </div>
  );
}
