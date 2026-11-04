package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kami2news/pos-server/models"
)

// POST /api/business-days  → 営業開始
func (h *Handler) OpenBusinessDay(c *gin.Context) {
	storeID, _ := c.Get("store_id")

	// 既にアクティブな営業日があれば返す（べき等）
	var existing models.BusinessDay
	if err := h.db.Get(&existing, `
		SELECT * FROM business_days WHERE store_id=$1 AND is_active=TRUE`, storeID); err == nil {
		c.JSON(http.StatusOK, existing)
		return
	}

	var bd models.BusinessDay
	if err := h.db.Get(&bd, `
		INSERT INTO business_days (store_id) VALUES ($1) RETURNING *`, storeID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "営業日開始失敗"})
		return
	}
	c.JSON(http.StatusCreated, bd)
}

// PATCH /api/business-days/close  → 営業終了
func (h *Handler) CloseBusinessDay(c *gin.Context) {
	storeID, _ := c.Get("store_id")

	var bd models.BusinessDay
	if err := h.db.Get(&bd, `
		UPDATE business_days
		SET is_active=FALSE, closed_at=NOW()
		WHERE store_id=$1 AND is_active=TRUE
		RETURNING *`, storeID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "アクティブな営業日がありません"})
		return
	}
	c.JSON(http.StatusOK, bd)
}

// GET /api/business-days/current  → 現在の営業日
func (h *Handler) CurrentBusinessDay(c *gin.Context) {
	storeID, _ := c.Get("store_id")

	var bd models.BusinessDay
	if err := h.db.Get(&bd, `
		SELECT * FROM business_days WHERE store_id=$1 AND is_active=TRUE`, storeID); err != nil {
		c.JSON(http.StatusOK, gin.H{"active": false})
		return
	}
	c.JSON(http.StatusOK, gin.H{"active": true, "business_day": bd})
}
