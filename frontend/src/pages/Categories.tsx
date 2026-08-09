import { useState, type FormEvent } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { IconPlus, IconTrash } from "@tabler/icons-react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { api } from "@/lib/api";
import type { Category } from "@/lib/types";

interface FlagState {
  has_warranty: boolean;
  has_expiry: boolean;
  can_sell: boolean;
  can_break: boolean;
  has_serial: boolean;
  has_model: boolean;
}

const EMPTY_FLAGS: FlagState = {
  has_warranty: false,
  has_expiry: false,
  can_sell: false,
  can_break: false,
  has_serial: false,
  has_model: false,
};

const FLAG_OPTIONS: { key: keyof FlagState; label: string; desc: string }[] = [
  { key: "has_warranty", label: "保修期", desc: "资产可填写保修月数，自动推算保修结束日期" },
  { key: "has_expiry", label: "到期日期", desc: "资产可填写到期日期，到期后自动标记已过期" },
  { key: "can_sell", label: "可售出", desc: "资产可标记已售出，填写售出日期与价格" },
  { key: "can_break", label: "可损坏", desc: "资产可标记已损坏，填写损坏日期" },
  { key: "has_serial", label: "序列号", desc: "资产需填写序列号" },
  { key: "has_model", label: "型号", desc: "资产需填写型号" },
];

function paramBadges(category: Category): string[] {
  const items: string[] = [];
  if (category.has_warranty) items.push("保修期");
  if (category.has_expiry) items.push("到期日期");
  if (category.can_sell) items.push("可售出");
  if (category.can_break) items.push("可损坏");
  if (category.has_serial) items.push("序列号");
  if (category.has_model) items.push("型号");
  return items;
}

export default function Categories() {
  const queryClient = useQueryClient();
  const { data: categories, isLoading, error } = useQuery({
    queryKey: ["categories"],
    queryFn: () => api<Category[]>("/api/categories"),
  });

  const [open, setOpen] = useState(false);
  const [editing, setEditing] = useState<Category | null>(null);
  const [name, setName] = useState("");
  const [flags, setFlags] = useState<FlagState>(EMPTY_FLAGS);
  const [formError, setFormError] = useState("");

  function openCreate() {
    setEditing(null);
    setName("");
    setFlags(EMPTY_FLAGS);
    setFormError("");
    setOpen(true);
  }

  function openEdit(category: Category) {
    setEditing(category);
    setName(category.name);
    setFlags({
      has_warranty: category.has_warranty,
      has_expiry: category.has_expiry,
      can_sell: category.can_sell,
      can_break: category.can_break,
      has_serial: category.has_serial,
      has_model: category.has_model,
    });
    setFormError("");
    setOpen(true);
  }

  const saveMutation = useMutation({
    mutationFn: () => {
      const payload = {
        name,
        ...flags,
      };
      return editing
        ? api<Category>(`/api/categories/${editing.id}`, { method: "PUT", body: JSON.stringify(payload) })
        : api<Category>("/api/categories", { method: "POST", body: JSON.stringify(payload) });
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["categories"] });
      setOpen(false);
    },
    onError: (err) => setFormError(err instanceof Error ? err.message : "保存失败"),
  });

  const deleteMutation = useMutation({
    mutationFn: (categoryId: number) =>
      api<{ ok: boolean }>(`/api/categories/${categoryId}`, { method: "DELETE" }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["categories"] }),
    onError: (err) => window.alert(err instanceof Error ? err.message : "删除失败"),
  });

  function toggleFlag(key: keyof FlagState, checked: boolean) {
    setFlags((f) => ({ ...f, [key]: checked }));
  }

  function submit(e: FormEvent) {
    e.preventDefault();
    if (!name.trim()) {
      setFormError("请输入类别名称");
      return;
    }
    saveMutation.mutate();
  }

  if (isLoading) return <p className="text-muted-foreground">加载中…</p>;
  if (error) return <p className="text-destructive">{error instanceof Error ? error.message : "加载失败"}</p>;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">类别</h1>
          <p className="text-sm text-muted-foreground">勾选类别拥有的参数，新建该类别资产时会要求填写对应内容</p>
        </div>
        <Button onClick={openCreate}>
          <IconPlus className="h-4 w-4" />
          新建类别
        </Button>
      </div>

      <div className="rounded-md border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>名称</TableHead>
              <TableHead>资产参数</TableHead>
              <TableHead className="text-right">资产数</TableHead>
              <TableHead className="w-24" />
            </TableRow>
          </TableHeader>
          <TableBody>
            {(categories ?? []).length === 0 && (
              <TableRow>
                <TableCell colSpan={4} className="py-10 text-center text-muted-foreground">
                  暂无类别，点击右上角「新建类别」开始配置
                </TableCell>
              </TableRow>
            )}
            {(categories ?? []).map((category) => {
              const badges = paramBadges(category);
              return (
                <TableRow key={category.id}>
                  <TableCell>
                    <div className="font-medium">{category.name}</div>
                  </TableCell>
                  <TableCell>
                    {badges.length === 0 ? (
                      <span className="text-muted-foreground">无</span>
                    ) : (
                      <div className="flex flex-wrap gap-1">
                        {badges.map((b) => (
                          <Badge key={b} variant="secondary">
                            {b}
                          </Badge>
                        ))}
                      </div>
                    )}
                  </TableCell>
                  <TableCell className="text-right">{category.assets_count}</TableCell>
                  <TableCell>
                    <div className="flex gap-1">
                      <Button variant="ghost" size="sm" onClick={() => openEdit(category)}>
                        编辑
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        className="text-destructive"
                        onClick={() => {
                          if (window.confirm(`确定删除类别「${category.name}」吗？`)) {
                            deleteMutation.mutate(category.id);
                          }
                        }}
                      >
                        <IconTrash className="h-4 w-4" />
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              );
            })}
          </TableBody>
        </Table>
      </div>

      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent className="max-w-xl">
          <DialogHeader>
            <DialogTitle>{editing ? "编辑类别" : "新建类别"}</DialogTitle>
          </DialogHeader>
          <form onSubmit={submit} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="cat-name">名称 *</Label>
              <Input id="cat-name" value={name} onChange={(e) => setName(e.target.value)} placeholder="如 数码产品" />
            </div>

            <div className="space-y-2">
              <Label>资产参数（勾选后新建该类资产时需要填写）</Label>
              <div className="rounded-md border p-4">
                <div className="space-y-3">
                  {FLAG_OPTIONS.map((option) => (
                    <label key={option.key} className="flex items-start gap-3">
                      <Checkbox
                        checked={flags[option.key]}
                        onCheckedChange={(v) => toggleFlag(option.key, v === true)}
                        className="mt-0.5"
                      />
                      <div>
                        <p className="text-sm font-medium">{option.label}</p>
                        <p className="text-xs text-muted-foreground">{option.desc}</p>
                      </div>
                    </label>
                  ))}
                </div>
              </div>
            </div>

            {formError && <p className="text-sm text-destructive">{formError}</p>}
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => setOpen(false)}>
                取消
              </Button>
              <Button type="submit" disabled={saveMutation.isPending}>
                保存
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </div>
  );
}
