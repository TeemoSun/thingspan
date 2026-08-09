import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { IconLayoutGrid, IconLayoutList, IconPlus, IconSearch } from "@tabler/icons-react";

import { StatusBadge } from "@/components/StatusBadge";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
} from "@/components/ui/card";
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
import { usePageTitle } from "@/lib/hooks";
import { AssetIcon } from "@/lib/icons";
import { cn } from "@/lib/utils";
import { STATUS_LABELS, type AssetList, type AssetStatus, type Category } from "@/lib/types";

type SortField = "purchase_date" | "purchase_price" | "daily_cost";

interface SortState {
  by: SortField;
  dir: "asc" | "desc";
}

function SortableHead({
  field,
  label,
  sort,
  onToggle,
  className,
  align = "left",
}: {
  field: SortField;
  label: string;
  sort: SortState | null;
  onToggle: (field: SortField) => void;
  className?: string;
  align?: "left" | "right";
}) {
  const active = sort?.by === field;
  return (
    <TableHead className={className}>
      <button
        type="button"
        onClick={() => onToggle(field)}
        className={cn(
          "inline-flex w-full select-none items-center gap-1 hover:text-foreground",
          align === "right" ? "justify-end" : "justify-start",
          active && "text-foreground"
        )}
      >
        {label}
        {active && <span aria-hidden>{sort!.dir === "asc" ? "↑" : "↓"}</span>}
      </button>
    </TableHead>
  );
}

export default function Assets() {
  usePageTitle("资产");
  const navigate = useNavigate();
  const [search, setSearch] = useState("");
  const [debounced, setDebounced] = useState("");
  const [categoryId, setCategoryId] = useState("");
  const [status, setStatus] = useState("");
  const [sort, setSort] = useState<SortState | null>({ by: "purchase_date", dir: "desc" });
  const [view, setView] = useState<"list" | "grid">(
    () => (localStorage.getItem("thingspan_assets_view") as "list" | "grid") ?? "list"
  );

  useEffect(() => {
    localStorage.setItem("thingspan_assets_view", view);
  }, [view]);

  useEffect(() => {
    const timer = setTimeout(() => setDebounced(search), 300);
    return () => clearTimeout(timer);
  }, [search]);

  function toggleSort(by: SortField) {
    setSort((prev) => {
      if (!prev || prev.by !== by) return { by, dir: "asc" };
      if (prev.dir === "asc") return { by, dir: "desc" };
      return null;
    });
  }

  const params = new URLSearchParams();
  if (debounced) params.set("search", debounced);
  if (categoryId) params.set("category_id", categoryId);
  if (status) params.set("status", status);
  if (sort) {
    params.set("sort_by", sort.by);
    params.set("sort_dir", sort.dir);
  }
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
        <Button
          variant={view === "list" ? "default" : "outline"}
          size="icon"
          className="ml-auto"
          onClick={() => setView("list")}
          title="列表视图"
        >
          <IconLayoutList className="h-4 w-4" />
        </Button>
        <Button
          variant={view === "grid" ? "default" : "outline"}
          size="icon"
          onClick={() => setView("grid")}
          title="卡片视图"
        >
          <IconLayoutGrid className="h-4 w-4" />
        </Button>
      </div>

      {isLoading && <p className="text-muted-foreground">加载中…</p>}
      {error && <p className="text-destructive">{error instanceof Error ? error.message : "加载失败"}</p>}

      {data && view === "list" && (
        <div className="rounded-md border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>名称</TableHead>
                <TableHead>类别</TableHead>
                <TableHead>状态</TableHead>
                <SortableHead field="purchase_date" label="购买日期" sort={sort} onToggle={toggleSort} />
                <SortableHead field="purchase_price" label="价格" sort={sort} onToggle={toggleSort} />
                <SortableHead
                  field="daily_cost"
                  label="日均成本"
                  sort={sort}
                  onToggle={toggleSort}
                  className="text-right"
                  align="right"
                />
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
                    <div className="flex items-center gap-3">
                      <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-muted">
                        <AssetIcon name={asset.icon} size={20} className="text-foreground" />
                      </div>
                      <div>
                        <div className="font-medium">{asset.name}</div>
                        {asset.brand || asset.model ? (
                          <div className="text-xs text-muted-foreground">
                            {[asset.brand, asset.model].filter(Boolean).join(" · ")}
                          </div>
                        ) : null}
                      </div>
                    </div>
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

      {data && view === "grid" && (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
          {data.items.length === 0 && (
            <p className="col-span-full py-10 text-center text-muted-foreground">
              暂无资产，点击右上角「新建资产」开始记录
            </p>
          )}
          {data.items.map((asset) => (
            <Card
              key={asset.id}
              className="cursor-pointer transition-shadow hover:shadow-md"
              onClick={() => navigate(`/assets/${asset.id}`)}
            >
              <CardContent className="flex flex-col gap-3 p-4">
                <div className="flex items-start justify-between gap-2">
                  <div className="flex items-center gap-3">
                    <div className="flex h-12 w-12 shrink-0 items-center justify-center rounded-lg bg-muted">
                      <AssetIcon name={asset.icon} size={26} className="text-foreground" />
                    </div>
                    <div className="min-w-0">
                      <div className="truncate font-medium">{asset.name}</div>
                      {asset.brand || asset.model ? (
                        <div className="truncate text-xs text-muted-foreground">
                          {[asset.brand, asset.model].filter(Boolean).join(" · ")}
                        </div>
                      ) : null}
                    </div>
                  </div>
                  <StatusBadge status={asset.status} />
                </div>
                <div className="text-xs text-muted-foreground">{asset.category_name}</div>
                <div className="flex items-end justify-between border-t pt-3">
                  <div className="text-xs text-muted-foreground">
                    购买于 {fmtDate(asset.purchase_date)}
                  </div>
                  <div className="text-right">
                    <div className="font-medium">{fmtMoneyShort(asset.purchase_price)}</div>
                    <div className="text-xs text-muted-foreground">
                      日均 {fmtMoney(asset.cost.daily_cost)}
                    </div>
                  </div>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      )}
    </div>
  );
}
