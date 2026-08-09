"""图标目录接口：返回全部 Tabler 图标名（从本服务加载，不使用 CDN）。"""
import json
from pathlib import Path

from fastapi import APIRouter, Depends

from app.api.deps import require_auth

router = APIRouter(prefix="/api/icons", tags=["icons"], dependencies=[Depends(require_auth)])

_ICONS: list[str] = json.loads((Path(__file__).resolve().parent.parent / "icon_names.json").read_text())["icons"]


@router.get("")
def list_icons() -> dict:
    return {"icons": _ICONS}
