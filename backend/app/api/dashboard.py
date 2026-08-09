"""仪表盘汇总接口。"""
from datetime import timedelta

from fastapi import APIRouter, Depends
from sqlalchemy import select
from sqlalchemy.orm import Session

from app.api.deps import require_auth
from app.database import get_db, local_now
from app.models import Asset, AssetStatus
from app.schemas import DashboardOut, ExpiringAsset
from app.services.cost import calc_cost, reminder_base_date

router = APIRouter(prefix="/api/dashboard", tags=["dashboard"], dependencies=[Depends(require_auth)])


@router.get("", response_model=DashboardOut)
def dashboard(db: Session = Depends(get_db)) -> DashboardOut:
    today = local_now().date()
    assets = db.scalars(select(Asset)).all()

    total_invested = round(sum(a.purchase_price for a in assets), 2)
    daily_total = round(sum(calc_cost(a, today).daily_cost for a in assets), 2)
    in_use = sum(1 for a in assets if a.status == AssetStatus.in_use.value)

    horizon = today + timedelta(days=30)
    expiring: list[ExpiringAsset] = []
    for a in assets:
        if a.status != AssetStatus.in_use.value:
            continue
        base = reminder_base_date(a)
        if base is None or not (today <= base <= horizon):
            continue
        date_type = "warranty" if a.warranty_end_date == base else "expiry"
        expiring.append(
            ExpiringAsset(
                id=a.id,
                name=a.name,
                category_name=a.category.name if a.category else "",
                target_date=base,
                days_left=(base - today).days,
                date_type=date_type,
            )
        )
    expiring.sort(key=lambda e: e.days_left)

    return DashboardOut(
        total_assets=len(assets),
        in_use_assets=in_use,
        total_invested=total_invested,
        daily_cost_total=daily_total,
        expiring_soon=expiring,
    )
