import jwt
from fastapi import Depends, HTTPException, status
from fastapi.security import HTTPAuthorizationCredentials, HTTPBearer

from app.security import decode_token

bearer_scheme = HTTPBearer(auto_error=False)


def require_auth(credentials: HTTPAuthorizationCredentials | None = Depends(bearer_scheme)) -> None:
    if credentials is None:
        raise HTTPException(status_code=status.HTTP_401_UNAUTHORIZED, detail="未登录")
    try:
        decode_token(credentials.credentials, "access")
    except (jwt.PyJWTError, HTTPException):
        raise HTTPException(status_code=status.HTTP_401_UNAUTHORIZED, detail="登录已过期")

