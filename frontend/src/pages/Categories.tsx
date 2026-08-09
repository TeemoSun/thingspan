import { useState, type FormEvent } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { IconPlus, IconTrash } from "@tabler/icons-react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
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
import {
  FIELD_TYPE_LABELS,
  TEMPLATE_LABELS,
  type Category,

  type FieldType,
  type Template,
} from "@/lib/types";

interface FieldRow {
  key: string;
  name: string;
  type: FieldType;
}

const TEMPLATE_DESCRIPTIONS: Record<Template, string> = {
  product: "数码类：支持品牌/型号/序列号/保修结束日期，可标记售出或损坏",
  membership: "会员类：支持到期日期，到期后自动标记已过期",
  other: "通用：仅基本信息，可标记售出或损坏",
};

export default function Categories() {
  const queryClient = useQueryClient();
  const { data: categories, isLoading, error } = useQuery({
    queryKey: ["categories"],
    queryFn: () => api<Category[]>("/api/categories"),
  });

  const [open, setOpen] = useState(false);
  const [editing, setEditing] = useState<Category | null>(null);
  const [name, setName] = useState("");
  const [icon, setIcon] = useState("tag");
  const [template, setTemplate] = useState<Template>("other");
  const [warrantyMonths, setWarrantyMonths] = useState("");
  const [fields, setFields] = useState<FieldRow[]>([]);
  const [formError, setFormError] = useState("");

  function openCreate() {
    setEditing(null);
    setName("");
    setIcon("tag");
    setTemplate("other");
    setWarrantyMonths("");
    setFields([]);
    setFormError("");
    setOpen(true);
  }

  function openEdit(category: Category) {
    setEditing(category);
    setName(category.name);
    setIcon(category.icon);
    setTemplate(category.template);
    setWarrantyMonths(category.warranty_months ? String(category.warranty_months) : "");
    setFields(category.fields.map((f) => ({ ...f })));
    setFormError("");
    setOpen(true);
  }

  const saveMutation = useMutation({
    mutationFn: () => {
      const payload = {
        name,
        icon,
        template,
        warranty_months: template === "product" && warrantyMonths ? parseInt(warrantyMonths, 10) : null,
        fields: fields
          .filter((f) => f.key.trim() && f.name.trim())
          .map((f) => ({ key: f.key.trim(), name: f.name.trim(), type: f.type })),
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

  function updateField(index: number, patch: Partial<FieldRow>) {
    setFields((list) => list.map((f, i) => (i === index ? { ...f, ...patch } : f)));
  }

  function submit(e: FormEvent) {
    e.preventDefault();
    if (!name.trim()) {
      setFormError("请输入类别名称");
      return;
    }
    if (fields.some((f) => f.key && !/^[a-z_][a-z0-9_]*$/.test(f.key))) {
      setFormError("字段 key 只允许小写字母、数字和下划线，且必须以字母或下划线开头");
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
          <p className="text-sm text-muted-foreground">每种类别可配置不同的模板与自定义字段</p>
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
              <TableHead>模板</TableHead>
              <TableHead>自定义字段</TableHead>
              <TableHead className="text-right">资产数</TableHead>
              <TableHead className="w-24" />
            </TableRow>
          </TableHeader>
          <TableBody>
            {(categories ?? []).length === 0 && (
              <TableRow>
                <TableCell colSpan={5} className="py-10 text-center text-muted-foreground">
                  暂无类别，点击右上角「新建类别」开始配置
                </TableCell>
              </TableRow>
            )}
            {(categories ?? []).map((category) => (
              <TableRow key={category.id}>
                <TableCell>
                  <div className="flex items-center gap-2 font-medium">
                    <Badge variant="outline" className="font-mono">
                      {category.icon}
                    </Badge>
                    {category.name}
                  </div>
                </TableCell>
                <TableCell>
                  <Badge variant="secondary">{TEMPLATE_LABELS[category.template]}</Badge>
                </TableCell>
                <TableCell className="text-muted-foreground">
                  {category.fields.length === 0
                    ? "-"
                    : category.fields
                        .map((f) => `${f.name}（${FIELD_TYPE_LABELS[f.type]}）`)
                        .join("、")}
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
            ))}
          </TableBody>
        </Table>
      </div>

      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent className="max-w-xl">
          <DialogHeader>
            <DialogTitle>{editing ? "编辑类别" : "新建类别"}</DialogTitle>
          </DialogHeader>
          <form onSubmit={submit} className="space-y-4">
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="cat-name">名称 *</Label>
                <Input id="cat-name" value={name} onChange={(e) => setName(e.target.value)} placeholder="如 数码产品" />
              </div>
              <div className="space-y-2">
                <Label htmlFor="cat-icon">图标（Tabler 名称）</Label>
                <Input id="cat-icon" value={icon} onChange={(e) => setIcon(e.target.value)} placeholder="如 device-mobile" />
              </div>
              <div className="space-y-2">
                <Label>模板</Label>
                <Select value={template} onValueChange={(v) => setTemplate(v as Template)}>
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {(Object.keys(TEMPLATE_LABELS) as Template[]).map((t) => (
                      <SelectItem key={t} value={t}>
                        {TEMPLATE_LABELS[t]}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                <p className="text-xs text-muted-foreground">{TEMPLATE_DESCRIPTIONS[template]}</p>
              </div>
              {template === "product" && (
                <div className="space-y-2">
                  <Label htmlFor="warranty">保修月数</Label>
                  <Input
                    id="warranty"
                    type="number"
                    min="1"
                    max="240"
                    value={warrantyMonths}
                    onChange={(e) => setWarrantyMonths(e.target.value)}
                    placeholder="如 12"
                  />
                  <p className="text-xs text-muted-foreground">按购买日自动推算保修结束日期</p>
                </div>
              )}
            </div>

            <div className="space-y-2">
              <Label>自定义字段</Label>
              <div className="space-y-2">
                {fields.map((field, index) => (
                  <div key={index} className="flex items-center gap-2">
                    <Input
                      className="w-40 font-mono"
                      placeholder="key，如 cpu"
                      value={field.key}
                      onChange={(e) => updateField(index, { key: e.target.value })}
                    />
                    <Input
                      className="flex-1"
                      placeholder="显示名称，如 CPU"
                      value={field.name}
                      onChange={(e) => updateField(index, { name: e.target.value })}
                    />
                    <Select
                      value={field.type}
                      onValueChange={(v) => updateField(index, { type: v as FieldType })}
                    >
                      <SelectTrigger className="w-24">
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        {(Object.keys(FIELD_TYPE_LABELS) as FieldType[]).map((t) => (
                          <SelectItem key={t} value={t}>
                            {FIELD_TYPE_LABELS[t]}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon"
                      className="text-muted-foreground"
                      onClick={() => setFields((list) => list.filter((_, i) => i !== index))}
                    >
                      <IconTrash className="h-4 w-4" />
                    </Button>
                  </div>
                ))}
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  onClick={() => setFields((list) => [...list, { key: "", name: "", type: "text" }])}
                >
                  <IconPlus className="h-4 w-4" />
                  添加字段
                </Button>
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
