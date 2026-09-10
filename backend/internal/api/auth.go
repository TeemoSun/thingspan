package api

import (
	"encoding/json"
	"net"
	"net/http"

	"thingspan/internal/models"
	"thingspan/internal/security"
)

type AuthHandler struct {
	pm   *security.PasswordManager
	jwtm *security.JWTManager
	rl   *security.RateLimiter
}

func NewAuthHandler(pm *security.PasswordManager, jwtm *security.JWTManager, rl *security.RateLimiter) *AuthHandler {
	return &AuthHandler{
		pm:   pm,
		jwtm: jwtm,
		rl:   rl,
	}
}

func getClientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && host != "" {
		return host
	}
	return "unknown"
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	ip := getClientIP(r)
	if !h.rl.CheckLimit(ip) {
		WriteError(w, http.StatusTooManyRequests, "登录失败次数过多，请稍后再试")
		return
	}

	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "无效的请求格式")
		return
	}

	if !h.pm.Verify(req.Password) {
		h.rl.RecordFailure(ip)
		WriteError(w, http.StatusUnauthorized, "密码错误")
		return
	}

	accessToken, err := h.jwtm.CreateAccessToken()
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Token 生成失败")
		return
	}
	refreshToken, err := h.jwtm.CreateRefreshToken()
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Token 生成失败")
		return
	}

	WriteJSON(w, http.StatusOK, models.TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "bearer",
	})
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req models.RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.RefreshToken == "" {
		WriteError(w, http.StatusBadRequest, "无效的请求格式")
		return
	}

	_, err := h.jwtm.DecodeToken(req.RefreshToken, "refresh")
	if err != nil {
		WriteError(w, http.StatusUnauthorized, "登录已过期，请重新登录")
		return
	}

	h.jwtm.RevokeRefreshToken(req.RefreshToken)

	accessToken, err := h.jwtm.CreateAccessToken()
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Token 生成失败")
		return
	}
	refreshToken, err := h.jwtm.CreateRefreshToken()
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Token 生成失败")
		return
	}

	WriteJSON(w, http.StatusOK, models.TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "bearer",
	})
}

