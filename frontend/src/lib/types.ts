export type AssetStatus = "in_use" | "sold" | "broken" | "expired";

export interface Category {
  id: number;
  name: string;
  has_warranty: boolean;
  has_expiry: boolean;
  can_sell: boolean;
  can_break: boolean;
  has_serial: boolean;
  has_model: boolean;
  assets_count: number;
}

export interface CostInfo {
  period_days: number;
  total_cost: number;
  daily_cost: number;
  formula: string;
}

export interface Asset {
  id: number;
  category_id: number;
  category_name: string;
  name: string;
  icon: string | null;
  brand: string | null;
  model: string | null;
  serial_number: string | null;
  purchase_date: string;
  purchase_price: number;
  warranty_months: number | null;
  warranty_end_date: string | null;
  expiry_date: string | null;
  status: AssetStatus;
  sale_date: string | null;
  sale_price: number | null;
  broken_date: string | null;
  notes: string | null;
  created_at?: string;
  updated_at?: string;
  cost: CostInfo;
}

export interface AssetCreatePayload {
  category_id: number;
  name: string;
  icon?: string | null;
  brand?: string | null;
  model?: string | null;
  serial_number?: string | null;
  purchase_date: string;
  purchase_price: number;
  warranty_months?: number | null;
  expiry_date?: string | null;
  notes?: string | null;
}

export interface AssetUpdatePayload extends Partial<AssetCreatePayload> {
  status?: AssetStatus;
  sale_date?: string | null;
  sale_price?: number | null;
  broken_date?: string | null;
}

export interface CategoryPayload {
  name: string;
  has_warranty: boolean;
  has_expiry: boolean;
  can_sell: boolean;
  can_break: boolean;
  has_serial: boolean;
  has_model: boolean;
}


export interface AssetList {
  items: Asset[];
  total: number;
}

export interface ExpiringAsset {
  id: number;
  name: string;
  category_name: string;
  target_date: string;
  days_left: number;
  date_type: "warranty" | "expiry";
}

export interface Dashboard {
  total_assets: number;
  in_use_assets: number;
  total_invested: number;
  daily_cost_total: number;
  expiring_soon: ExpiringAsset[];
}

export interface ReminderLog {
  id: number;
  asset_id: number;
  asset_name: string;
  target_date: string;
  lead_days: number;
  sent_at: string;
  sent: boolean;
  dismissed: boolean;
}

export const STATUS_LABELS: Record<AssetStatus, string> = {
  in_use: "使用中",
  sold: "已售出",
  broken: "已损坏",
  expired: "已过期",
};
