"""密码校验（bcrypt）与 JWT 双 Token 签发/校验。"""
import secrets
from datetime import datetime, timedelta, timezone
from typing import Any

import bcrypt
import jwt

from app.config import settings

_password_hash: bytes | None = None
# 已吊销的 refresh token jti -> 过期时间（进程内，单用户足够）
_revoked: dict[str, datetime] = {}


def init_password() -> None:
    global _password_hash
    if _password_hash is None:
        _password_hash = bcrypt.hashpw(settings.app_password.encode(), bcrypt.gensalt())


def verify_password(password: str) -> bool:
    init_password()
    return bcrypt.checkpw(password.encode(), _password_hash or b"")


def _create_token(token_type: str, expire: timedelta) -> str:
    now = datetime.now(timezone.utc)
    payload: dict[str, Any] = {
        "sub": "user",
        "type": token_type,
        "iat": now,
        "exp": now + expire,
        "jti": secrets.token_hex(8),
    }
    return jwt.encode(payload, settings.jwt_secret, algorithm="HS256")


def create_access_token() -> str:
    return _create_token("access", timedelta(minutes=settings.access_token_expire_minutes))


def create_refresh_token() -> str:
    return _create_token("refresh", timedelta(days=settings.refresh_token_expire_days))


def decode_token(token: str, expected_type: str) -> dict:
    payload = jwt.decode(token, settings.jwt_secret, algorithms=["HS256"])
    if payload.get("type") != expected_type:
        raise jwt.InvalidTokenError("wrong token type")
    if expected_type == "refresh":
        _cleanup_revoked()
        if payload.get("jti") in _revoked:
            raise jwt.InvalidTokenError("token revoked")
    return payload


def revoke_refresh(token: str) -> None:
    try:
        payload = jwt.decode(token, settings.jwt_secret, algorithms=["HS256"])
        if payload.get("type") == "refresh":
            _revoked[payload["jti"]] = datetime.fromtimestamp(payload["exp"], tz=timezone.utc)
    except jwt.PyJWTError:
        pass


def _cleanup_revoked() -> None:
    now = datetime.now(timezone.utc)
    for jti in [j for j, exp in _revoked.items() if exp <= now]:
        _revoked.pop(jti, None)
