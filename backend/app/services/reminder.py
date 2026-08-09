"""到期提醒扫描：按档位窗口发送邮件（失败自动重试）、会员到期自动标记。"""
import logging
from datetime import date, timedelta

from sqlalchemy import select

from app.config import settings
from app.database import SessionLocal, local_now
from app.models import Asset, AssetStatus, Category, ReminderLog, Template
from app.services.cost import reminder_base_date
from app.services.email import send_email

logger = logging.getLogger(__name__)


def today_local() -> date:
    return local_now().date()


def _email_body(asset: Asset, target_date: date, days_left: int) -> str:
    when = f"{days_left} 天后到期" if days_left > 0 else "已到期"
    return (
        f"【资产到期提醒】\n\n"
        f"资产：{asset.name}\n"
        f"类别：{asset.category.name if asset.category else '-'}\n"
        f"到期日期：{target_date.isoformat()}\n"
        f"距离：{when}\n\n"
        f"请及时处理。"
    )


async def run_reminder_scan() -> None:
    """每日任务：在提前天数窗口内发送提醒（每档一次，失败次日自动重试）；会员到期自动标记。"""
    today = today_local()
    with SessionLocal() as session:
        # 1. 发送提醒：基准日进入某档窗口即发送该档，每档最多一次、每资产每天最多一封；
        #    发送失败记录 sent=False，次日自动重试
        assets = session.scalars(
            select(Asset).where(Asset.status == AssetStatus.in_use.value)
        ).all()
        for asset in assets:
            base = reminder_base_date(asset)
            if base is None or base < today:
                continue
            for lead in sorted(settings.lead_days):
                if base > today + timedelta(days=lead):
                    continue
                log = session.scalar(
                    select(ReminderLog).where(
                        ReminderLog.asset_id == asset.id,
                        ReminderLog.target_date == base,
                        ReminderLog.lead_days == lead,
                    )
                )
                if log and log.sent:
                    break
                days_left = (base - today).days
                ok = await send_email(
                    f"【Thingspan】{asset.name} {days_left} 天后到期",
                    _email_body(asset, base, days_left),
                )
                if log:
                    if ok:
                        log.sent = True
                        session.commit()
                else:
                    session.add(
                        ReminderLog(asset_id=asset.id, target_date=base, lead_days=lead, sent=ok)
                    )
                    session.commit()
                break

        # 2. 会员到期自动标记为已过期
        overdue = session.scalars(
            select(Asset)
            .join(Category, Asset.category_id == Category.id)
            .where(
                Asset.status == AssetStatus.in_use.value,
                Category.template == Template.membership.value,
                Asset.expiry_date.is_not(None),
                Asset.expiry_date < today,
            )
        ).all()
        for asset in overdue:
            asset.status = AssetStatus.expired.value
            logger.info("资产自动标记为已过期: %s", asset.name)
        if overdue:
            session.commit()
