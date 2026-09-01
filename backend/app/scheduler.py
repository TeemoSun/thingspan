"""APScheduler 定时任务。"""
import asyncio
import logging

from apscheduler.schedulers.asyncio import AsyncIOScheduler
from apscheduler.triggers.cron import CronTrigger

from app.config import settings
from app.services.reminder import run_reminder_scan

logger = logging.getLogger(__name__)

scheduler = AsyncIOScheduler(timezone=settings.tz)
_background_tasks: set[asyncio.Task] = set()


def start_scheduler() -> None:
    scheduler.add_job(
        run_reminder_scan,
        CronTrigger(hour=settings.reminder_check_hour, minute=0),
        id="reminder-scan",
        max_instances=1,
        coalesce=True,
        misfire_grace_time=3600,
    )
    scheduler.start()
    logger.info("调度器已启动，每日 %s:00 检查提醒", settings.reminder_check_hour)
    # 启动时补跑一次，处理漏发与到期标记（保持强引用防止被 GC）
    task = asyncio.create_task(run_reminder_scan())
    _background_tasks.add(task)
    task.add_done_callback(_background_tasks.discard)

