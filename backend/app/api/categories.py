"""类别管理接口。"""
from fastapi import APIRouter, Depends, HTTPException
from sqlalchemy import func, select
from sqlalchemy.orm import Session

from app.api.deps import require_auth
from app.database import get_db
from app.models import Asset, Category
from app.schemas import CategoryCreate, CategoryOut, CategoryUpdate

router = APIRouter(prefix="/api/categories", tags=["categories"], dependencies=[Depends(require_auth)])


def _to_out(category: Category, assets_count: int) -> CategoryOut:
    return CategoryOut(
        id=category.id,
        name=category.name,
        has_warranty=category.has_warranty,
        has_expiry=category.has_expiry,
        can_sell=category.can_sell,
        can_break=category.can_break,
        has_serial=category.has_serial,
        has_model=category.has_model,
        warranty_months=category.warranty_months,
        assets_count=assets_count,
    )


def _apply(body: CategoryCreate | CategoryUpdate, category: Category) -> None:
    category.name = body.name
    category.has_warranty = body.has_warranty
    category.has_expiry = body.has_expiry
    category.can_sell = body.can_sell
    category.can_break = body.can_break
    category.has_serial = body.has_serial
    category.has_model = body.has_model
    category.warranty_months = body.warranty_months if body.has_warranty else None


@router.get("", response_model=list[CategoryOut])
def list_categories(db: Session = Depends(get_db)) -> list[CategoryOut]:
    categories = db.scalars(select(Category).order_by(Category.created_at)).all()
    counts = dict(db.execute(select(Asset.category_id, func.count()).group_by(Asset.category_id)).all())
    return [_to_out(c, counts.get(c.id, 0)) for c in categories]


@router.post("", response_model=CategoryOut)
def create_category(body: CategoryCreate, db: Session = Depends(get_db)) -> CategoryOut:
    if db.scalar(select(Category).where(Category.name == body.name)):
        raise HTTPException(status_code=400, detail="类别名称已存在")
    category = Category()
    _apply(body, category)
    db.add(category)
    db.commit()
    db.refresh(category)
    return _to_out(category, 0)


@router.put("/{category_id}", response_model=CategoryOut)
def update_category(category_id: int, body: CategoryUpdate, db: Session = Depends(get_db)) -> CategoryOut:
    category = db.get(Category, category_id)
    if not category:
        raise HTTPException(status_code=404, detail="类别不存在")
    dup = db.scalar(select(Category).where(Category.name == body.name, Category.id != category_id))
    if dup:
        raise HTTPException(status_code=400, detail="类别名称已存在")
    _apply(body, category)
    db.commit()
    db.refresh(category)
    return _to_out(category, db.scalar(select(func.count()).select_from(Asset).where(Asset.category_id == category_id)))


@router.delete("/{category_id}")
def delete_category(category_id: int, db: Session = Depends(get_db)) -> dict:
    category = db.get(Category, category_id)
    if not category:
        raise HTTPException(status_code=404, detail="类别不存在")
    count = db.scalar(select(func.count()).select_from(Asset).where(Asset.category_id == category_id))
    if count:
        raise HTTPException(status_code=400, detail="该类别下还有资产，无法删除")
    db.delete(category)
    db.commit()
    return {"ok": True}
