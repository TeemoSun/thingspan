"""数据库连接与会话管理。"""
from datetime import datetime
from zoneinfo import ZoneInfo

from sqlalchemy import create_engine
from sqlalchemy.orm import DeclarativeBase, sessionmaker

from app.config import settings


class Base(DeclarativeBase):
    pass


def local_now() -> datetime:
    """返回配置时区的当前时间（naive，统一本地时间存储）。"""
    return datetime.now(ZoneInfo(settings.tz)).replace(tzinfo=None)


settings.data_dir.mkdir(parents=True, exist_ok=True)
engine = create_engine(
    f"sqlite:///{settings.database_path}",
    connect_args={"check_same_thread": False},
)
SessionLocal = sessionmaker(bind=engine, autoflush=False, expire_on_commit=False)


def get_db():
    db = SessionLocal()
    try:
        yield db
    finally:
        db.close()
