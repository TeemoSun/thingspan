"""SMTP 邮件发送。"""
import logging
import smtplib
import ssl
from email.message import EmailMessage

import anyio

from app.config import settings

logger = logging.getLogger(__name__)


def _send_sync(subject: str, body: str) -> bool:
    if not settings.smtp_host or not settings.smtp_user or not settings.mail_to:
        logger.warning("SMTP 未配置，跳过邮件: %s", subject)
        return False
    msg = EmailMessage()
    msg["Subject"] = subject
    msg["From"] = settings.smtp_user
    msg["To"] = settings.mail_to
    msg.set_content(body)
    ctx = ssl.create_default_context()
    try:
        if settings.smtp_port == 465:
            server = smtplib.SMTP_SSL(settings.smtp_host, settings.smtp_port, context=ctx, timeout=30)
        else:
            server = smtplib.SMTP(settings.smtp_host, settings.smtp_port, timeout=30)
            server.starttls(context=ctx)
        with server:
            server.login(settings.smtp_user, settings.smtp_password)
            server.send_message(msg)
        logger.info("邮件已发送: %s", subject)
        return True
    except Exception:
        logger.exception("邮件发送失败: %s", subject)
        return False


async def send_email(subject: str, body: str) -> bool:
    return await anyio.to_thread.run_sync(_send_sync, subject, body)
