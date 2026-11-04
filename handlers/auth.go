package handlers

import (
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/kami2news/pos-server/middleware"
	"github.com/kami2news/pos-server/models"
)

// ─────────────────────────────────────────────
// POST /auth/register  （管理者のみ・店舗作成）
// Header: X-Admin-Secret: <ADMIN_SECRET env>
// ─────────────────────────────────────────────
type registerReq struct {
	StoreName string `json:"store_name" binding:"required"`
	Password  string `json:"password"   binding:"required,min=4"`
}

func (h *Handler) Register(c *gin.Context) {
	// 管理者シークレット確認
	if c.GetHeader("X-Admin-Secret") != os.Getenv("ADMIN_SECRET") {
		c.JSON(http.StatusForbidden, gin.H{"error": "管理者権限が必要です"})
		return
	}

	var req registerReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "入力が不正です"})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ハッシュ生成失敗"})
		return
	}

	var store models.Store
	if err := h.db.Get(&store, `
		INSERT INTO stores (name, password_hash) VALUES ($1, $2)
		RETURNING id, name, created_at`,
		req.StoreName, string(hash)); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "既に存在する店舗名です"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":         store.ID,
		"store_name": store.Name,
		"message":    "店舗を作成しました",
	})
}

// ─────────────────────────────────────────────
// POST /auth/login
// ─────────────────────────────────────────────
type loginReq struct {
	StoreName string `json:"store_name" binding:"required"`
	Password  string `json:"password"   binding:"required"`
}

func (h *Handler) Login(c *gin.Context) {
	var req loginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "入力が不正です"})
		return
	}

	var store models.Store
	if err := h.db.Get(&store, `SELECT * FROM stores WHERE name = $1`, req.StoreName); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "店舗名またはパスワードが違います"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(store.PasswordHash), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "店舗名またはパスワードが違います"})
		return
	}

	token, err := generateToken(store.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "トークン生成失敗"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token":      token,
		"store_id":   store.ID,
		"store_name": store.Name,
	})
}

// ─────────────────────────────────────────────
// GET /auth/me
// ─────────────────────────────────────────────
func (h *Handler) Me(c *gin.Context) {
	storeID, _ := c.Get("store_id")
	var store models.Store
	if err := h.db.Get(&store, `
		SELECT id, name, created_at FROM stores WHERE id = $1`, storeID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "店舗が見つかりません"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"id":         store.ID,
		"store_name": store.Name,
	})
}

func generateToken(storeID string) (string, error) {
	claims := &middleware.Claims{
		StoreID: storeID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(30 * 24 * time.Hour)), // 30日
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(os.Getenv("JWT_SECRET")))
}
