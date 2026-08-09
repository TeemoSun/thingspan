"""SQLAlchemy 数据模型：Category / Asset / ReminderLog。"""
import enum
from datetime import date, datetime

from sqlalchemy import Boolean, Date, DateTime, Float, ForeignKey, Integer, String, Text, UniqueConstraint
from sqlalchemy.orm import Mapped, mapped_column, relationship

from app.database import Base, local_now


class AssetStatus(str, enum.Enum):
    in_use = "in_use"
    sold = "sold"
    broken = "broken"
    expired = "expired"


class Category(Base):
    """类别：以勾选方式配置资产参数（保修期/到期日期/售出/损坏/序列号/型号）。"""

    __tablename__ = "categories"

    id: Mapped[int] = mapped_column(primary_key=True)
    name: Mapped[str] = mapped_column(String(100), unique=True)
    # 资产参数勾选：勾选后新建该类资产时表单出现对应字段/状态
    has_warranty: Mapped[bool] = mapped_column(Boolean, default=False)  # 保修期（资产填写保修月数，自动推算保修结束日期）
    has_expiry: Mapped[bool] = mapped_column(Boolean, default=False)  # 到期日期（到期后自动标记已过期）
    can_sell: Mapped[bool] = mapped_column(Boolean, default=False)  # 可售出（可标记已售出，填写售出日期/价格）
    can_break: Mapped[bool] = mapped_column(Boolean, default=False)  # 可损坏（可标记已损坏，填写损坏日期）
    has_serial: Mapped[bool] = mapped_column(Boolean, default=False)  # 序列号
    has_model: Mapped[bool] = mapped_column(Boolean, default=False)  # 型号
    created_at: Mapped[datetime] = mapped_column(DateTime, default=local_now)

    assets: Mapped[list["Asset"]] = relationship(back_populates="category")


class Asset(Base):
    __tablename__ = "assets"

    id: Mapped[int] = mapped_column(primary_key=True)
    category_id: Mapped[int] = mapped_column(ForeignKey("categories.id"))
    name: Mapped[str] = mapped_column(String(200))
    icon: Mapped[str | None] = mapped_column(String(50), nullable=True)
    brand: Mapped[str | None] = mapped_column(String(100), nullable=True)
    model: Mapped[str | None] = mapped_column(String(100), nullable=True)
    serial_number: Mapped[str | None] = mapped_column(String(100), nullable=True)
    purchase_date: Mapped[date] = mapped_column(Date)
    purchase_price: Mapped[float] = mapped_column(Float)
    warranty_months: Mapped[int | None] = mapped_column(Integer, nullable=True)
    warranty_end_date: Mapped[date | None] = mapped_column(Date, nullable=True)
    expiry_date: Mapped[date | None] = mapped_column(Date, nullable=True)
    status: Mapped[str] = mapped_column(String(20), default=AssetStatus.in_use.value)
    sale_date: Mapped[date | None] = mapped_column(Date, nullable=True)
    sale_price: Mapped[float | None] = mapped_column(Float, nullable=True)
    broken_date: Mapped[date | None] = mapped_column(Date, nullable=True)
    notes: Mapped[str | None] = mapped_column(Text, nullable=True)
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
