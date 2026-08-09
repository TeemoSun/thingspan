import dayjs from "dayjs";

export function fmtMoney(value: number | null | undefined): string {
  if (value === null || value === undefined) return "-";
  return "¥" + value.toLocaleString("zh-CN", { minimumFractionDigits: 2, maximumFractionDigits: 2 });
}

export function fmtMoneyShort(value: number | null | undefined): string {
  if (value === null || value === undefined) return "-";
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
