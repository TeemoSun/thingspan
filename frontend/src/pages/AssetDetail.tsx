import { useEffect, useMemo, useState, type FormEvent } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { IconArrowLeft, IconCalendarEvent, IconCoins, IconTag, IconTrash } from "@tabler/icons-react";

import { StatusBadge } from "@/components/StatusBadge";
import IconPicker from "@/components/IconPicker";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogDescription,
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
import { Textarea } from "@/components/ui/textarea";
import { api } from "@/lib/api";
import { fmtDate, fmtMoney, fmtMoneyShort, todayStr } from "@/lib/format";
import { usePageTitle } from "@/lib/hooks";
import { AssetIcon } from "@/lib/icons";
import { type Asset, type Category } from "@/lib/types";

interface FormState {
  name: string;
  icon: string;
  category_id: string;
  brand: string;
  model: string;
  serial_number: string;
  purchase_date: string;
  purchase_price: string;
  warranty_months: string;
  expiry_date: string;
  notes: string;
}

const FORMULA_LABELS: Record<string, string> = {
  in_use: "使用中（价格 ÷ 已用天数；有到期日按到期日计）",
  sold: "已售出（买入 − 卖出）÷ 持有天数",
  broken: "已损坏（价格 ÷ 使用天数）",
  expired: "已到期（价格 ÷ 有效天数）",
};

export default function AssetDetail() {
  const { id } = useParams();
  const isNew = id === "new" || id === undefined;
  const assetId = isNew ? null : Number(id);
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const { data: categories } = useQuery({
    queryKey: ["categories"],
    queryFn: () => api<Category[]>("/api/categories"),
  });
  const { data: asset, isLoading: assetLoading } = useQuery({
    queryKey: ["asset", assetId],
    queryFn: () => api<Asset>(`/api/assets/${assetId}`),
    enabled: !isNew,
  });
  usePageTitle(isNew ? "新建资产" : asset?.name || "资产详情");

  const [form, setForm] = useState<FormState>({
    name: "",
    icon: "",
    category_id: "",
    brand: "",
    model: "",
    serial_number: "",
    purchase_date: todayStr(),
    purchase_price: "",
    warranty_months: "",
    expiry_date: "",
    notes: "",
  });
  const [error, setError] = useState("");
  const [iconOpen, setIconOpen] = useState(false);
  const [sellOpen, setSellOpen] = useState(false);
  const [brokenOpen, setBrokenOpen] = useState(false);
  const [saleDate, setSaleDate] = useState(todayStr());
  const [salePrice, setSalePrice] = useState("");
  const [brokenDate, setBrokenDate] = useState(todayStr());

  useEffect(() => {
    if (!asset) return;
    setForm({
      name: asset.name,
      icon: asset.icon ?? "",
      category_id: String(asset.category_id),
      brand: asset.brand ?? "",
      model: asset.model ?? "",
      serial_number: asset.serial_number ?? "",
      purchase_date: asset.purchase_date,
      purchase_price: String(asset.purchase_price),
      warranty_months: asset.warranty_months ? String(asset.warranty_months) : "",
      expiry_date: asset.expiry_date ?? "",
      notes: asset.notes ?? "",
    });
  }, [asset]);

  const category = useMemo(
    () => (categories ?? []).find((c) => c.id === Number(form.category_id)),
    [categories, form.category_id]
  );

  function setField(field: keyof FormState, value: string) {
    setForm((f) => ({ ...f, [field]: value }));
  }

  async function submit(e: FormEvent) {
    e.preventDefault();
    setError("");
    if (!form.category_id) {
      setError("请选择类别");
      return;
    }
    const payload = {
      category_id: Number(form.category_id),
      name: form.name,
      icon: form.icon || null,
      brand: form.brand || null,
      model: form.model || null,
      serial_number: form.serial_number || null,
      purchase_date: form.purchase_date,
      purchase_price: parseFloat(form.purchase_price) || 0,
      warranty_months: form.warranty_months ? parseInt(form.warranty_months, 10) : null,
      expiry_date: form.expiry_date || null,
      notes: form.notes || null,
    };
    try {
      const saved = isNew
        ? await api<Asset>("/api/assets", { method: "POST", body: JSON.stringify(payload) })
        : await api<Asset>(`/api/assets/${assetId}`, { method: "PUT", body: JSON.stringify(payload) });
      await queryClient.invalidateQueries({ queryKey: ["assets"] });
      await queryClient.invalidateQueries({ queryKey: ["dashboard"] });
      navigate(`/assets/${saved.id}`);
    } catch (err) {
      setError(err instanceof Error ? err.message : "保存失败");
    }
  }

  const statusMutation = useMutation({
    mutationFn: (payload: Record<string, unknown>) =>
      api<Asset>(`/api/assets/${assetId}`, { method: "PUT", body: JSON.stringify(payload) }),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["asset", assetId] });
      await queryClient.invalidateQueries({ queryKey: ["assets"] });
      await queryClient.invalidateQueries({ queryKey: ["dashboard"] });
      setSellOpen(false);
      setBrokenOpen(false);
    },
    onError: (err) => setError(err instanceof Error ? err.message : "操作失败"),
  });

  const deleteMutation = useMutation({
    mutationFn: () => api<{ ok: boolean }>(`/api/assets/${assetId}`, { method: "DELETE" }),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["assets"] });
      await queryClient.invalidateQueries({ queryKey: ["dashboard"] });
      navigate("/assets");
    },
    onError: (err) => setError(err instanceof Error ? err.message : "删除失败"),
  });

  const showWarrantyField = category?.has_warranty === true;
  const showExpiryField = category?.has_expiry === true;
  const showModelField = category?.has_model === true;
  const showSerialField = category?.has_serial === true;
  const canSell = category?.can_sell === true;
  const canBreak = category?.can_break === true;

  const warrantyEndPreview = useMemo(() => {
    const months = parseInt(form.warranty_months, 10);
    if (!showWarrantyField || !form.purchase_date || !months || months < 1) return null;
    const d = new Date(`${form.purchase_date}T00:00:00`);
    d.setDate(d.getDate() + months * 30);
    const y = d.getFullYear();
    const m = String(d.getMonth() + 1).padStart(2, "0");
    const day = String(d.getDate()).padStart(2, "0");
    return `${y}-${m}-${day}`;
  }, [showWarrantyField, form.purchase_date, form.warranty_months]);

  if (assetLoading) return <p className="text-muted-foreground">加载中…</p>;

  return (
    <div className="mx-auto max-w-3xl space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div className="flex min-w-0 items-center gap-2">
          <Button variant="ghost" size="icon" onClick={() => navigate("/assets")}>
            <IconArrowLeft className="h-4 w-4" />
          </Button>
          <h1 className="flex min-w-0 items-center gap-2 text-2xl font-bold">
            {!isNew && asset?.icon && <AssetIcon name={asset.icon} className="h-5 w-5 shrink-0" />}
            <span className="truncate">{isNew ? "新建资产" : asset?.name}</span>
          </h1>
          {!isNew && asset && <StatusBadge status={asset.status} />}
        </div>
        {!isNew && (
          <Button
            variant="ghost"
            size="sm"
            className="text-destructive"
            onClick={() => {
              if (window.confirm(`确定删除「${asset?.name}」吗？此操作不可恢复。`)) deleteMutation.mutate();
            }}
          >
            <IconTrash className="h-4 w-4" />
            删除
          </Button>
        )}
      </div>

      {!isNew && asset && (
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="flex items-center gap-2 text-base">
              <IconCoins className="h-4 w-4 text-muted-foreground" />
              日均成本
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="flex flex-wrap items-end gap-6">
              <div>
                <div className="text-3xl font-bold">{fmtMoney(asset.cost.daily_cost)}</div>
                <p className="text-xs text-muted-foreground">每天</p>
              </div>
              <div className="space-y-1 text-sm text-muted-foreground">
                <p>累计成本：{fmtMoney(asset.cost.total_cost)}</p>
                <p>统计周期：{asset.cost.period_days} 天</p>
                <p>口径：{FORMULA_LABELS[asset.cost.formula] ?? asset.cost.formula}</p>
              </div>
            </div>
          </CardContent>
        </Card>
      )}

      {!isNew && asset && (
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="flex items-center gap-2 text-base">
              <IconCalendarEvent className="h-4 w-4 text-muted-foreground" />
              状态与日期
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            {asset.status === "sold" && (
              <p className="text-sm text-muted-foreground">
                已于 {fmtDate(asset.sale_date)} 售出，售价 {fmtMoneyShort(asset.sale_price)}
              </p>
            )}
            {asset.status === "broken" && (
              <p className="text-sm text-muted-foreground">已于 {fmtDate(asset.broken_date)} 损坏</p>
            )}
            {asset.status === "expired" && (
              <p className="text-sm text-muted-foreground">
                已于 {fmtDate(asset.expiry_date)} 到期
              </p>
            )}
            <div className="flex flex-wrap gap-2">
              {asset.status === "in_use" && (
                <>
                  {canSell && (
                    <Button variant="outline" size="sm" onClick={() => setSellOpen(true)}>
                      标记已售出
                    </Button>
                  )}
                  {canBreak && (
                    <Button variant="outline" size="sm" onClick={() => setBrokenOpen(true)}>
                      标记已损坏
                    </Button>
                  )}
                </>
              )}
              {(asset.status === "sold" || asset.status === "broken") && (
                <Button
                  variant="secondary"
                  size="sm"
                  onClick={() => statusMutation.mutate({ status: "in_use" })}
                  disabled={statusMutation.isPending}
                >
                  恢复使用中
                </Button>
              )}
            </div>
          </CardContent>
        </Card>
      )}

      <Card>
        <CardHeader className="pb-2">
          <CardTitle className="flex items-center gap-2 text-base">
            <IconTag className="h-4 w-4 text-muted-foreground" />
            基本信息
          </CardTitle>
          {!isNew && <CardDescription>编辑后点击保存生效</CardDescription>}
        </CardHeader>
        <CardContent>
          <form onSubmit={submit} className="space-y-4">
            <div className="grid gap-4 sm:grid-cols-2">
              <div className="space-y-2">
                <Label htmlFor="name">名称 *</Label>
                <Input id="name" value={form.name} onChange={(e) => setField("name", e.target.value)} required placeholder="如 iPhone 16 Pro" />
              </div>
              <div className="space-y-2 sm:col-span-2">
                <Label>图标</Label>
                <div className="flex items-center gap-2">
                  <Button type="button" variant="outline" onClick={() => setIconOpen(true)} className="gap-2">
                    <AssetIcon name={form.icon || null} size={16} />
                    {form.icon || "选择图标"}
                  </Button>
                  {form.icon && (
                    <Button type="button" variant="ghost" size="sm" onClick={() => setField("icon", "")}>
                      清除
                    </Button>
                  )}
                </div>
                <IconPicker
                  open={iconOpen}
                  onOpenChange={setIconOpen}
                  value={form.icon}
                  onSelect={(icon: string) => setField("icon", icon)}
                />
              </div>
              <div className="space-y-2">
                <Label>类别 *</Label>
                <Select value={form.category_id} onValueChange={(v) => setField("category_id", v)}>
                  <SelectTrigger>
                    <SelectValue placeholder="选择类别" />
                  </SelectTrigger>
                  <SelectContent>
                    {(categories ?? []).map((c) => (
                      <SelectItem key={c.id} value={String(c.id)}>
                        {c.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-2">
                <Label htmlFor="brand">品牌</Label>
                <Input id="brand" value={form.brand} onChange={(e) => setField("brand", e.target.value)} placeholder="如 Apple" />
              </div>
              {showModelField && (
                <div className="space-y-2">
                  <Label htmlFor="model">型号</Label>
                  <Input id="model" value={form.model} onChange={(e) => setField("model", e.target.value)} placeholder="如 A3101" />
                </div>
              )}
              {showSerialField && (
                <div className="space-y-2">
                  <Label htmlFor="serial">序列号</Label>
                  <Input id="serial" value={form.serial_number} onChange={(e) => setField("serial_number", e.target.value)} />
                </div>
              )}
              <div className="space-y-2">
                <Label htmlFor="purchase_date">购买日期 *</Label>
                <Input
                  id="purchase_date"
                  type="date"
                  value={form.purchase_date}
                  onChange={(e) => setField("purchase_date", e.target.value)}
                  required
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="purchase_price">购买价格（元）*</Label>
                <Input id="purchase_price" type="number" min="0" step="0.01" value={form.purchase_price} onChange={(e) => setField("purchase_price", e.target.value)} required placeholder="如 5999" />
              </div>
              {showWarrantyField && (
                <div className="space-y-2">
                  <Label htmlFor="warranty_months">保修期（月数）</Label>
                  <Input
                    id="warranty_months"
                    type="number"
                    min="1"
                    max="600"
                    value={form.warranty_months}
                    onChange={(e) => setField("warranty_months", e.target.value)}
                    placeholder="如 12"
                  />
                  {warrantyEndPreview && (
                    <p className="text-xs text-muted-foreground">保修至 {warrantyEndPreview}（购买日 + 月数×30 天自动推算）</p>
                  )}
                </div>
              )}
              {showExpiryField && (
                <div className="space-y-2">
                  <Label htmlFor="expiry">到期日期</Label>
                  <Input id="expiry" type="date" value={form.expiry_date} onChange={(e) => setField("expiry_date", e.target.value)} />
                </div>
              )}
              <div className="space-y-2 sm:col-span-2">
                <Label htmlFor="notes">备注</Label>
                <Textarea id="notes" value={form.notes} onChange={(e) => setField("notes", e.target.value)} placeholder="可选，记录购买渠道、凭证等" />
              </div>
            </div>
            {error && <p className="text-sm text-destructive">{error}</p>}
            <div className="flex justify-end gap-2">
              <Button type="button" variant="outline" onClick={() => navigate("/assets")}>
                取消
              </Button>
              <Button type="submit" disabled={statusMutation.isPending}>
                保存
              </Button>
            </div>
          </form>
        </CardContent>
      </Card>

      <Dialog open={sellOpen} onOpenChange={setSellOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>标记已售出</DialogTitle>
            <DialogDescription>填写售出日期与售价，日均成本将按持有期计算</DialogDescription>
          </DialogHeader>
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <div className="space-y-2">
              <Label>售出日期</Label>
              <Input type="date" value={saleDate} onChange={(e) => setSaleDate(e.target.value)} />
            </div>
            <div className="space-y-2">
              <Label>售出价格（元）</Label>
              <Input type="number" min="0" step="0.01" value={salePrice} onChange={(e) => setSalePrice(e.target.value)} placeholder="如 4000" />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setSellOpen(false)}>
              取消
            </Button>
            <Button
              disabled={!saleDate || !salePrice || statusMutation.isPending}
              onClick={() =>
                statusMutation.mutate({
                  status: "sold",
                  sale_date: saleDate,
                  sale_price: parseFloat(salePrice),
                })
              }
            >
              确认售出
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={brokenOpen} onOpenChange={setBrokenOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>标记已损坏</DialogTitle>
            <DialogDescription>填写损坏日期，日均成本将按截至损坏日的使用期计算</DialogDescription>
          </DialogHeader>
          <div className="space-y-2">
            <Label>损坏日期</Label>
            <Input type="date" value={brokenDate} onChange={(e) => setBrokenDate(e.target.value)} />
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setBrokenOpen(false)}>
              取消
            </Button>
            <Button
              disabled={!brokenDate || statusMutation.isPending}
              onClick={() => statusMutation.mutate({ status: "broken", broken_date: brokenDate })}
            >
              确认损坏
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
