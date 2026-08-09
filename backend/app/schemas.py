"""Pydantic 请求/响应模型。"""
from datetime import date, datetime
from typing import Literal

from pydantic import BaseModel, ConfigDict, Field, field_validator

from app.models import AssetStatus, Template

FieldType = Literal["text", "date", "number"]


class FieldDef(BaseModel):
    key: str = Field(pattern=r"^[a-z_][a-z0-9_]*$")
    name: str = Field(min_length=1, max_length=50)
    type: FieldType


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
    icon: str = Field(default="tag", max_length=50)
    template: Template = Template.other
    fields: list[FieldDef] = Field(default_factory=list)
    warranty_months: int | None = Field(default=None, ge=1, le=240)

    @field_validator("fields")
    @classmethod
    def unique_field_keys(cls, v: list[FieldDef]) -> list[FieldDef]:
        keys = [f.key for f in v]
        if len(keys) != len(set(keys)):
            raise ValueError("字段 key 不能重复")
        return v


class CategoryUpdate(CategoryCreate):
    pass


class CategoryOut(BaseModel):
    model_config = ConfigDict(from_attributes=True)

    id: int
    name: str
    icon: str
    template: str
    fields: list[dict]
    warranty_months: int | None
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
    warranty_end_date: date | None = None
    expiry_date: date | None = None
    notes: str | None = None
    custom_values: dict = Field(default_factory=dict)


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
    warranty_end_date: date | None = None
    expiry_date: date | None = None
    status: AssetStatus | None = None
    sale_date: date | None = None
    sale_price: float | None = Field(default=None, ge=0)
    broken_date: date | None = None
    notes: str | None = None
    custom_values: dict | None = None


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
    category_template: str = ""
    name: str
    brand: str | None
    model: str | None
    serial_number: str | None
    purchase_date: date
    purchase_price: float
    warranty_end_date: date | None
    expiry_date: date | None
    status: str
    sale_date: date | None
    sale_price: float | None
    broken_date: date | None
    notes: str | None
    custom_values: dict
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
