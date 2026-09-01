import dayjs from "dayjs";

export function fmtMoney(value: number | null | undefined): string {
  if (value === null || value === undefined || typeof value !== "number" || !Number.isFinite(value)) return "-";
  return "¥" + value.toLocaleString("zh-CN", { minimumFractionDigits: 2, maximumFractionDigits: 2 });
}

export function fmtMoneyShort(value: number | null | undefined): string {
  if (value === null || value === undefined || typeof value !== "number" || !Number.isFinite(value)) return "-";
  return "¥" + value.toLocaleString("zh-CN", { maximumFractionDigits: 2 });
}

export function fmtDate(value: string | null | undefined): string {
  return value ?? "-";
}

export function fmtDateTime(value: string): string {
  return dayjs(value).format("YYYY-MM-DD HH:mm");
}

export function todayStr(): string {
  return dayjs().format("YYYY-MM-DD");
}

export function calcWarrantyEndDate(purchaseDate: string, months: number | null | undefined): string | null {
  if (!purchaseDate || !months || months < 1) return null;
  const d = dayjs(purchaseDate);
  if (!d.isValid()) return null;
  return d.add(months * 30, "day").format("YYYY-MM-DD");
}

