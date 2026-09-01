from fastapi import APIRouter, Depends, HTTPException
from sqlalchemy import select
from sqlalchemy.orm import Session, joinedload

from app.api.deps import require_auth
from app.database import get_db
from app.models import ReminderLog
from app.schemas import ReminderOut

router = APIRouter(prefix="/api/reminders", tags=["reminders"], dependencies=[Depends(require_auth)])


def _to_out(log: ReminderLog) -> ReminderOut:
    return ReminderOut(
        id=log.id,
        asset_id=log.asset_id,
        asset_name=log.asset.name if log.asset else "",
        target_date=log.target_date,
        lead_days=log.lead_days,
        sent_at=log.sent_at,
        sent=log.sent,
        dismissed=log.dismissed,
    )


@router.get("", response_model=list[ReminderOut])
def list_reminders(db: Session = Depends(get_db)) -> list[ReminderOut]:
    logs = db.scalars(
        select(ReminderLog).options(joinedload(ReminderLog.asset)).order_by(ReminderLog.sent_at.desc())
    ).all()
    return [_to_out(l) for l in logs]



@router.post("/{reminder_id}/dismiss", response_model=ReminderOut)
def dismiss_reminder(reminder_id: int, db: Session = Depends(get_db)) -> ReminderOut:
    log = db.get(ReminderLog, reminder_id)
    if not log:
        raise HTTPException(status_code=404, detail="提醒记录不存在")
    log.dismissed = True
    db.commit()
    db.refresh(log)
    return _to_out(log)
