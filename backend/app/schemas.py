"""Pydantic 请求/响应模型。"""
from datetime import date, datetime

from pydantic import BaseModel, ConfigDict, Field, field_validator

from app.models import AssetStatus

# ---------- 认证 ----------
class LoginRequest(BaseModel):
    password: str


class TokenResponse(BaseModel):
    access_token: str
    refresh_token: str
    token_type: str = "bearer"


class RefreshRequest(BaseModel):
    refresh_token: str


# ---------- 类别 ----------
class CategoryCreate(BaseModel):
    name: str = Field(min_length=1, max_length=50)
    has_warranty: bool = False
    has_expiry: bool = False
    can_sell: bool = False
    can_break: bool = False
    has_serial: bool = False
    has_model: bool = False


class CategoryUpdate(CategoryCreate):
    pass


class CategoryOut(BaseModel):
    model_config = ConfigDict(from_attributes=True)

    id: int
    name: str
    has_warranty: bool
    has_expiry: bool
    can_sell: bool
    can_break: bool
    has_serial: bool
    has_model: bool
    assets_count: int = 0


# ---------- 资产 ----------
class AssetBase(BaseModel):
    category_id: int
    name: str = Field(min_length=1, max_length=100)
    brand: str | None = None
    model: str | None = None
    serial_number: str | None = None
    purchase_date: date
    purchase_price: float = Field(ge=0)
    warranty_months: int | None = Field(default=None, ge=1, le=600)
    warranty_end_date: date | None = None
    expiry_date: date | None = None
    notes: str | None = None


class AssetCreate(AssetBase):
    status: AssetStatus = AssetStatus.in_use
    sale_date: date | None = None
    sale_price: float | None = Field(default=None, ge=0)
    broken_date: date | None = None

    @field_validator("purchase_price")
    @classmethod
    def positive_price(cls, v: float) -> float:
        return round(v, 2)


class AssetUpdate(BaseModel):
    category_id: int | None = None
    name: str | None = Field(default=None, min_length=1, max_length=100)
    brand: str | None = None
    model: str | None = None
    serial_number: str | None = None
    purchase_date: date | None = None
    purchase_price: float | None = Field(default=None, ge=0)
    warranty_months: int | None = Field(default=None, ge=1, le=600)
    warranty_end_date: date | None = None
    expiry_date: date | None = None
    status: AssetStatus | None = None
    sale_date: date | None = None
    sale_price: float | None = Field(default=None, ge=0)
    broken_date: date | None = None
    notes: str | None = None


class CostOut(BaseModel):
    period_days: int
    total_cost: float
    daily_cost: float
    formula: str


class AssetOut(BaseModel):
    model_config = ConfigDict(from_attributes=True)

    id: int
    category_id: int
    category_name: str = ""
    name: str
    brand: str | None
    model: str | None
    serial_number: str | None
    purchase_date: date
    purchase_price: float
    warranty_months: int | None
    warranty_end_date: date | None
    expiry_date: date | None
    status: str
    sale_date: date | None
    sale_price: float | None
    broken_date: date | None
    notes: str | None
    created_at: datetime
    updated_at: datetime
    cost: CostOut | None = None


class AssetListOut(BaseModel):
    items: list[AssetOut]
    total: int


# ---------- 仪表盘 ----------
class ExpiringAsset(BaseModel):
    id: int
    name: str
    category_name: str
    target_date: date
    days_left: int
    date_type: str  # warranty | expiry


class DashboardOut(BaseModel):
    total_assets: int
    in_use_assets: int
    total_invested: float
    daily_cost_total: float
    expiring_soon: list[ExpiringAsset]


# ---------- 提醒 ----------
class ReminderOut(BaseModel):
    model_config = ConfigDict(from_attributes=True)

    id: int
    asset_id: int
    asset_name: str = ""
    target_date: date
    lead_days: int
    sent_at: datetime
    sent: bool = True
    dismissed: bool
