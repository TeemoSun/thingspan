"""应用配置，全部来自环境变量 / .env 文件。"""
from pathlib import Path

from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    model_config = SettingsConfigDict(env_file=".env", env_file_encoding="utf-8", extra="ignore")

    # 认证
    app_password: str = ""
    jwt_secret: str = ""
    access_token_expire_minutes: int = 30
    refresh_token_expire_days: int = 30

    # 时区与数据
    tz: str = "Asia/Shanghai"
    data_dir: Path = Path("./data")

    # 邮件提醒
    smtp_host: str = ""
    smtp_port: int = 465
    smtp_user: str = ""
    smtp_password: str = ""
    mail_to: str = ""

    # 提醒策略
    reminder_lead_days: str = "30,7,1"
    reminder_check_hour: int = 9

    @property
    def database_path(self) -> Path:
        return self.data_dir / "thingspan.db"

    @property
    def lead_days(self) -> list[int]:
        days = []
        for part in self.reminder_lead_days.split(","):
            part = part.strip()
            if part.isdigit():
                days.append(int(part))
        return sorted(days)


settings = Settings()
