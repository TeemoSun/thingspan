"""认证接口：登录 / 刷新 Token。"""
from collections import deque
from time import monotonic

from fastapi import APIRouter, Depends, HTTPException, Request

from app.api.deps import require_auth
from app.schemas import LoginRequest, RefreshRequest, TokenResponse
from app.security import create_access_token, create_refresh_token, decode_token, revoke_refresh, verify_password

router = APIRouter(prefix="/api/auth", tags=["auth"])

LOGIN_RATE_LIMIT = 5
LOGIN_RATE_WINDOW = 60.0
_login_attempts: dict[str, deque[float]] = {}


def _check_login_rate(ip: str) -> None:
    now = monotonic()
    window_start = now - LOGIN_RATE_WINDOW
    recent = _login_attempts.setdefault(ip, deque())
    while recent and recent[0] <= window_start:
        recent.popleft()
    if len(recent) >= LOGIN_RATE_LIMIT:
        raise HTTPException(status_code=429, detail="登录尝试过于频繁，请稍后再试")
    recent.append(now)
    if len(_login_attempts) > 1024:
        for key, vals in list(_login_attempts.items()):
            if not vals or vals[-1] <= window_start:
                del _login_attempts[key]


@router.post("/login", response_model=TokenResponse)
def login(body: LoginRequest, request: Request) -> TokenResponse:
    _check_login_rate(request.client.host if request.client else "unknown")
    if not verify_password(body.password):
        raise HTTPException(status_code=401, detail="密码错误")
    return TokenResponse(access_token=create_access_token(), refresh_token=create_refresh_token())


@router.post("/refresh", response_model=TokenResponse, dependencies=[Depends(require_auth)])
def refresh(body: RefreshRequest) -> TokenResponse:
    try:
        decode_token(body.refresh_token, "refresh")
    except Exception:
        raise HTTPException(status_code=401, detail="登录已过期，请重新登录")
    revoke_refresh(body.refresh_token)
    return TokenResponse(access_token=create_access_token(), refresh_token=create_refresh_token())
