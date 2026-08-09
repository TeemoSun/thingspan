"""SQLAlchemy 数据模型：Category / Asset / ReminderLog。"""
import enum
from datetime import date, datetime

from sqlalchemy import JSON, Boolean, Date, DateTime, Float, ForeignKey, Integer, String, Text, UniqueConstraint
from sqlalchemy.orm import Mapped, mapped_column, relationship

from app.database import Base, local_now


class Template(str, enum.Enum):
    product = "product"
    membership = "membership"
    other = "other"


class AssetStatus(str, enum.Enum):
    in_use = "in_use"
    sold = "sold"
    broken = "broken"
    expired = "expired"


class Category(Base):
    __tablename__ = "categories"

    id: Mapped[int] = mapped_column(primary_key=True)
    name: Mapped[str] = mapped_column(String(100), unique=True)
    icon: Mapped[str] = mapped_column(String(50), default="tag")
    template: Mapped[str] = mapped_column(String(20), default=Template.other.value)
    # 自定义字段定义: [{"key": "cpu", "name": "CPU", "type": "text"|"date"|"number"}]
    fields: Mapped[list] = mapped_column(JSON, default=list)
    # 仅 product 模板：保修月数，购买日 + 月数自动推算保修结束日期
    warranty_months: Mapped[int | None] = mapped_column(Integer, nullable=True)
    created_at: Mapped[datetime] = mapped_column(DateTime, default=local_now)

    assets: Mapped[list["Asset"]] = relationship(back_populates="category")


class Asset(Base):
    __tablename__ = "assets"

    id: Mapped[int] = mapped_column(primary_key=True)
    category_id: Mapped[int] = mapped_column(ForeignKey("categories.id"))
    name: Mapped[str] = mapped_column(String(200))
    brand: Mapped[str | None] = mapped_column(String(100), nullable=True)
    model: Mapped[str | None] = mapped_column(String(100), nullable=True)
    serial_number: Mapped[str | None] = mapped_column(String(100), nullable=True)
    purchase_date: Mapped[date] = mapped_column(Date)
    purchase_price: Mapped[float] = mapped_column(Float)
    warranty_end_date: Mapped[date | None] = mapped_column(Date, nullable=True)
    expiry_date: Mapped[date | None] = mapped_column(Date, nullable=True)
    status: Mapped[str] = mapped_column(String(20), default=AssetStatus.in_use.value)
    sale_date: Mapped[date | None] = mapped_column(Date, nullable=True)
    sale_price: Mapped[float | None] = mapped_column(Float, nullable=True)
    broken_date: Mapped[date | None] = mapped_column(Date, nullable=True)
    notes: Mapped[str | None] = mapped_column(Text, nullable=True)
    custom_values: Mapped[dict] = mapped_column(JSON, default=dict)
    created_at: Mapped[datetime] = mapped_column(DateTime, default=local_now)
    updated_at: Mapped[datetime] = mapped_column(DateTime, default=local_now, onupdate=local_now)

    category: Mapped[Category] = relationship(back_populates="assets")


class ReminderLog(Base):
    __tablename__ = "reminder_logs"
    __table_args__ = (UniqueConstraint("asset_id", "target_date", "lead_days", name="uq_reminder_log"),)

    id: Mapped[int] = mapped_column(primary_key=True)
    asset_id: Mapped[int] = mapped_column(ForeignKey("assets.id"))
    target_date: Mapped[date] = mapped_column(Date)
    lead_days: Mapped[int] = mapped_column(Integer)
    sent_at: Mapped[datetime] = mapped_column(DateTime, default=local_now)
    sent: Mapped[bool] = mapped_column(Boolean, default=True)
    dismissed: Mapped[bool] = mapped_column(Boolean, default=False)

    asset: Mapped[Asset] = relationship()
