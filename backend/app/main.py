"""FastAPI 应用入口：迁移、认证初始化、调度器、静态托管。"""
import logging
from contextlib import asynccontextmanager
from pathlib import Path

from alembic import command
from alembic.config import Config as AlembicConfig
from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import FileResponse

from app.api import assets, auth, categories, dashboard, icons, reminders
from app.config import settings
from app.scheduler import scheduler
from app.security import init_password

logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(name)s: %(message)s")
logger = logging.getLogger(__name__)

BACKEND_DIR = Path(__file__).resolve().parent.parent
STATIC_DIR = BACKEND_DIR / "app" / "static"


def run_migrations() -> None:
    cfg = AlembicConfig(str(BACKEND_DIR / "alembic.ini"))
    cfg.set_main_option("script_location", str(BACKEND_DIR / "alembic"))
    command.upgrade(cfg, "head")
    logger.info("数据库迁移完成")


def validate_secrets() -> None:
    if not settings.app_password or settings.app_password in ("admin", "change-me"):
        raise RuntimeError(
            "APP_PASSWORD 未配置或仍为默认值，请设置 .env 中的 APP_PASSWORD 后重启"
        )
    if not settings.jwt_secret or settings.jwt_secret == "change-me":
        raise RuntimeError(
            "JWT_SECRET 未配置或仍为默认值，请用 `openssl rand -hex 32` 生成并写入 .env 后重启"
        )


@asynccontextmanager
async def lifespan(app: FastAPI):
    validate_secrets()
    settings.data_dir.mkdir(parents=True, exist_ok=True)
    run_migrations()
    init_password()
    from app.scheduler import start_scheduler

    start_scheduler()
    logger.info("Thingspan 启动完成")
    yield
    scheduler.shutdown(wait=False)


app = FastAPI(title="Thingspan", version="0.1.0", lifespan=lifespan)

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=False,
    allow_methods=["*"],
    allow_headers=["*"],
)

app.include_router(auth.router)
app.include_router(categories.router)
app.include_router(assets.router)
app.include_router(dashboard.router)
app.include_router(icons.router)
app.include_router(reminders.router)


@app.get("/api/health")
def health() -> dict:
    return {"ok": True}


if (STATIC_DIR / "index.html").exists():
    @app.get("/{full_path:path}", include_in_schema=False)
    async def spa_fallback(full_path: str):
        if full_path.startswith("api/"):
            from fastapi import HTTPException

            raise HTTPException(status_code=404, detail="Not Found")
        candidate = STATIC_DIR / "assets" / full_path.removeprefix("assets/") if full_path.startswith("assets/") else STATIC_DIR / full_path
        if candidate.is_file():
            return FileResponse(candidate)
        return FileResponse(STATIC_DIR / "index.html")
