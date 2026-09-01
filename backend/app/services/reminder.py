import logging
from datetime import date, timedelta

from sqlalchemy import select
from sqlalchemy.orm import joinedload

from app.config import settings
from app.database import SessionLocal, local_now
from app.models import Asset, AssetStatus, Category, ReminderLog
from app.services.cost import reminder_target_dates
from app.services.email import send_email

logger = logging.getLogger(__name__)


def today_local() -> date:
    return local_now().date()


def _email_body(asset: Asset, target_date: date, days_left: int, date_type: str = "expiry") -> str:
    type_label = "保修" if date_type == "warranty" else "资产"
    when = f"{days_left} 天后到期" if days_left > 0 else "已到期"
    return (
        f"【{type_label}到期提醒】\n\n"
        f"资产：{asset.name}\n"
        f"类别：{asset.category.name if asset.category else '-'}\n"
        f"{type_label}日期：{target_date.isoformat()}\n"
        f"距离：{when}\n\n"
        f"请及时处理。"
    )


async def run_reminder_scan() -> None:
    """每日任务：在提前天数窗口内发送提醒（每档一次，失败次日自动重试）；到期自动标记。"""
    today = today_local()
    with SessionLocal() as session:
        # 1. 发送提醒：基准日进入某档窗口即发送该档，每档最多一次、每资产每天最多一封；
        #    发送失败记录 sent=False，次日自动重试
        try:
            assets = session.scalars(
                select(Asset)
                .options(joinedload(Asset.category))
                .where(Asset.status == AssetStatus.in_use.value)
            ).all()
        except Exception:
            logger.exception("获取待提醒资产列表失败")
            assets = []

        for asset in assets:
            try:
                targets = reminder_target_dates(asset, today)
                if not targets:
                    continue
                # 遍历资产的有效未来目标日期（保修日 / 到期日）
                sent_today = False
                for date_type, base in targets:
                    if sent_today:
                        break
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
                        type_name = "保修" if date_type == "warranty" else "到期"
                        ok = await send_email(
                            f"【Thingspan】{asset.name} {type_name} {days_left} 天后到期",
                            _email_body(asset, base, days_left, date_type),
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
                        sent_today = True
                        break
            except Exception:
                logger.exception("处理资产提醒失败: asset_id=%s", asset.id)
                session.rollback()

        # 2. 勾选了「到期日期」类别的资产，到期自动标记为已过期
        try:
            overdue = session.scalars(
                select(Asset)
                .join(Category, Asset.category_id == Category.id)
                .where(
                    Asset.status == AssetStatus.in_use.value,
                    Category.has_expiry.is_(True),
                    Asset.expiry_date.is_not(None),
                    Asset.expiry_date < today,
                )
            ).all()
            for asset in overdue:
                asset.status = AssetStatus.expired.value
                logger.info("资产自动标记为已过期: %s", asset.name)
            if overdue:
                session.commit()
        except Exception:
            logger.exception("自动标记过期资产失败")
            session.rollback()

