"""日均成本计算。"""
from dataclasses import dataclass
from datetime import date

from app.models import Asset, AssetStatus


@dataclass
class CostInfo:
    period_days: int
    total_cost: float
    daily_cost: float
    formula: str


def calc_cost(asset: Asset, today: date) -> CostInfo:
    """按状态计算日均成本。

    - 使用中: 购买价 / 自购买起天数
    - 已售出: (购买价 - 售出价) / (售出日 - 购买日)
    - 已损坏: 购买价 / (损坏日 - 购买日)
    - 已过期: 购买价 / (到期日 - 购买日)
    """
    formula = AssetStatus.in_use.value
    total_cost = asset.purchase_price

    if asset.status == AssetStatus.sold.value and asset.sale_date and asset.sale_price is not None:
        days = (asset.sale_date - asset.purchase_date).days
        total_cost = round(asset.purchase_price - asset.sale_price, 2)
        formula = AssetStatus.sold.value
    elif asset.status == AssetStatus.broken.value and asset.broken_date:
        days = (asset.broken_date - asset.purchase_date).days
        formula = AssetStatus.broken.value
    elif asset.status == AssetStatus.expired.value and asset.expiry_date:
        days = (asset.expiry_date - asset.purchase_date).days
        formula = AssetStatus.expired.value
    else:
        days = (today - asset.purchase_date).days
        formula = AssetStatus.in_use.value

    daily_cost = round(total_cost / days, 4) if days > 0 else 0.0
    return CostInfo(period_days=max(days, 0), total_cost=total_cost, daily_cost=daily_cost, formula=formula)


def sync_expiry_status(asset: Asset, today: date) -> bool:
    """勾选「到期日期」类别的资产，按到期日自动同步状态：过期则标记已过期，日期改到未来则恢复使用中。"""
    if asset.category is None or not asset.category.has_expiry:
        return False
    if asset.status == AssetStatus.in_use.value and asset.expiry_date is not None and asset.expiry_date < today:
        asset.status = AssetStatus.expired.value
        return True
    if asset.status == AssetStatus.expired.value and (asset.expiry_date is None or asset.expiry_date >= today):
        asset.status = AssetStatus.in_use.value
        return True
    return False


def reminder_base_date(asset: Asset) -> date | None:
    """提醒基准日期：保修结束日期或到期日期，取非空者（优先保修）。"""
    return asset.warranty_end_date or asset.expiry_date
