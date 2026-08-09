"""资产管理接口。"""
from dataclasses import asdict
from datetime import date, timedelta

from fastapi import APIRouter, Depends, HTTPException, Query
from sqlalchemy import or_, select
from sqlalchemy.orm import Session

from app.api.deps import require_auth
from app.database import get_db, local_now
from app.models import Asset, AssetStatus, Category
from app.schemas import AssetCreate, AssetListOut, AssetOut, AssetUpdate
from app.services.cost import calc_cost

router = APIRouter(prefix="/api/assets", tags=["assets"], dependencies=[Depends(require_auth)])


def _to_out(asset: Asset) -> AssetOut:
    today = local_now().date()
    return AssetOut(
        id=asset.id,
        category_id=asset.category_id,
        category_name=asset.category.name if asset.category else "",
        name=asset.name,
        brand=asset.brand,
        model=asset.model,
        serial_number=asset.serial_number,
        purchase_date=asset.purchase_date,
        purchase_price=asset.purchase_price,
        warranty_end_date=asset.warranty_end_date,
        expiry_date=asset.expiry_date,
        status=asset.status,
        sale_date=asset.sale_date,
        sale_price=asset.sale_price,
        broken_date=asset.broken_date,
        notes=asset.notes,
        created_at=asset.created_at,
        updated_at=asset.updated_at,
        cost=asdict(calc_cost(asset, today)),
    )


def _validate_category(db: Session, category_id: int) -> Category:
    category = db.get(Category, category_id)
    if not category:
        raise HTTPException(status_code=400, detail="类别不存在")
    return category


def _apply_warranty(asset: Asset, category: Category, force: bool = False) -> None:
    """类别勾选了保修期且配置了保修月数时，自动推算保修结束日期（用户手动填过且未改购买日则不覆盖）。"""
    if category.has_warranty and category.warranty_months and asset.purchase_date:
        if force or asset.warranty_end_date is None:
            asset.warranty_end_date = asset.purchase_date + timedelta(days=category.warranty_months * 30)


def _check_status_fields(category: Category, status: str, sale_date, sale_price, broken_date, *, check_flags: bool = True) -> None:
    if status == AssetStatus.sold.value:
        if check_flags and not category.can_sell:
            raise HTTPException(status_code=400, detail="该类别未勾选「可售出」，无法标记已售出")
        if sale_date is None or sale_price is None:
            raise HTTPException(status_code=400, detail="标记已售出需填写售出日期和售出价格")
    if status == AssetStatus.broken.value:
        if check_flags and not category.can_break:
            raise HTTPException(status_code=400, detail="该类别未勾选「可损坏」，无法标记已损坏")
        if broken_date is None:
            raise HTTPException(status_code=400, detail="标记已损坏需填写损坏日期")


@router.get("", response_model=AssetListOut)
def list_assets(
    db: Session = Depends(get_db),
    category_id: int | None = None,
    status: str | None = None,
    search: str | None = Query(default=None, max_length=100),
) -> AssetListOut:
    query = select(Asset)
    if category_id:
        query = query.where(Asset.category_id == category_id)
    if status:
        query = query.where(Asset.status == status)
    if search:
        like = f"%{search}%"
        query = query.where(
            or_(
                Asset.name.like(like),
                Asset.brand.like(like),
                Asset.model.like(like),
                Asset.serial_number.like(like),
            )
        )
    query = query.order_by(Asset.created_at.desc())
    items = db.scalars(query).all()
    return AssetListOut(items=[_to_out(a) for a in items], total=len(items))


@router.post("", response_model=AssetOut)
def create_asset(body: AssetCreate, db: Session = Depends(get_db)) -> AssetOut:
    category = _validate_category(db, body.category_id)
    _check_status_fields(category, body.status.value, body.sale_date, body.sale_price, body.broken_date)
    asset = Asset(
        category_id=body.category_id,
        name=body.name,
        brand=body.brand,
        model=body.model,
        serial_number=body.serial_number,
        purchase_date=body.purchase_date,
        purchase_price=body.purchase_price,
        warranty_end_date=body.warranty_end_date,
        expiry_date=body.expiry_date,
        status=body.status.value,
        sale_date=body.sale_date,
        sale_price=body.sale_price,
        broken_date=body.broken_date,
        notes=body.notes,
    )
    _apply_warranty(asset, category)
    db.add(asset)
    db.commit()
    db.refresh(asset)
    return _to_out(asset)


@router.get("/{asset_id}", response_model=AssetOut)
def get_asset(asset_id: int, db: Session = Depends(get_db)) -> AssetOut:
    asset = db.get(Asset, asset_id)
    if not asset:
        raise HTTPException(status_code=404, detail="资产不存在")
    return _to_out(asset)


@router.put("/{asset_id}", response_model=AssetOut)
def update_asset(asset_id: int, body: AssetUpdate, db: Session = Depends(get_db)) -> AssetOut:
    asset = db.get(Asset, asset_id)
    if not asset:
        raise HTTPException(status_code=404, detail="资产不存在")

    data = body.model_dump(exclude_unset=True)

    new_category_id = data.get("category_id", asset.category_id)
    category = _validate_category(db, new_category_id)

    status = data.get("status", asset.status)
    if isinstance(status, AssetStatus):
        status = status.value
    sale_date = data.get("sale_date", asset.sale_date)
    sale_price = data.get("sale_price", asset.sale_price)
    broken_date = data.get("broken_date", asset.broken_date)

    if "status" in data and status != asset.status:
        _check_status_fields(category, status, sale_date, sale_price, broken_date)
        if status != AssetStatus.sold.value:
            asset.sale_date = None
            asset.sale_price = None
        if status != AssetStatus.broken.value:
            asset.broken_date = None
        asset.status = status

    for field in ("category_id", "name", "brand", "model", "serial_number", "purchase_date", "purchase_price",
                  "warranty_end_date", "expiry_date", "notes"):
        if field in data:
            setattr(asset, field, data[field])

    if asset.status == AssetStatus.sold.value:
        if "sale_date" in data:
            asset.sale_date = sale_date
        if "sale_price" in data:
            asset.sale_price = sale_price
        _check_status_fields(category, asset.status, asset.sale_date, asset.sale_price, asset.broken_date, check_flags=False)
    if asset.status == AssetStatus.broken.value:
        if "broken_date" in data:
            asset.broken_date = broken_date
        _check_status_fields(category, asset.status, asset.sale_date, asset.sale_price, asset.broken_date, check_flags=False)

    force_warranty = "purchase_date" in data and "warranty_end_date" not in data
    _apply_warranty(asset, category, force=force_warranty)
    db.commit()
    db.refresh(asset)
    return _to_out(asset)


@router.delete("/{asset_id}")
def delete_asset(asset_id: int, db: Session = Depends(get_db)) -> dict:
    asset = db.get(Asset, asset_id)
    if not asset:
        raise HTTPException(status_code=404, detail="资产不存在")
    db.delete(asset)
    db.commit()
    return {"ok": True}
