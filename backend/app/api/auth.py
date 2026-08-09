"""认证接口：登录 / 刷新 Token。"""
from fastapi import APIRouter, Depends, HTTPException

from app.api.deps import require_auth
from app.schemas import LoginRequest, RefreshRequest, TokenResponse
from app.security import create_access_token, create_refresh_token, decode_token, revoke_refresh, verify_password

router = APIRouter(prefix="/api/auth", tags=["auth"])


@router.post("/login", response_model=TokenResponse)
def login(body: LoginRequest) -> TokenResponse:
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
