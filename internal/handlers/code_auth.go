package handlers

import (
	"net/http"

	"authorization-server/internal/services"

	"github.com/gin-gonic/gin"
)

type CodeAuthHandler struct {
	authService *services.AuthService
}

func NewCodeAuthHandler(authService *services.AuthService) *CodeAuthHandler {
	return &CodeAuthHandler{authService: authService}
}

func (h *CodeAuthHandler) VerifyCode(c *gin.Context) {
	var request struct {
		Code string `json:"code" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Проверяем код авторизации
	valid, err := h.authService.VerifyAuthCode(request.Code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if !valid {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid or expired code"})
		return
	}

	// Получаем токены по коду - ИСПРАВЛЕНИЕ: получаем 2 значения, а не 3
	tokens, err := h.authService.GetTokensByCode(request.Code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  tokens.AccessToken,
		"refresh_token": tokens.RefreshToken,
	})
}
