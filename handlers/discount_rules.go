package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kami2news/pos-server/models"
	"github.com/lib/pq"
)

// GET /api/discount-rules
func (h *Handler) ListDiscountRules(c *gin.Context) {
	storeID, _ := c.Get("store_id")
	var rules []models.DiscountRule
	if err := h.db.Select(&rules, `
		SELECT * FROM discount_rules
		WHERE store_id=$1
		ORDER BY created_at`, storeID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "取得失敗"})
		return
	}
	if rules == nil {
		rules = []models.DiscountRule{}
	}
	c.JSON(http.StatusOK, rules)
}

type discountRuleReq struct {
	Name             string   `json:"name"               binding:"required"`
	IsActive         bool     `json:"is_active"`
	ConditionType    string   `json:"condition_type"     binding:"required"`
	TargetProductIDs []string `json:"target_product_ids"`
	MinQuantity      int      `json:"min_quantity"`
	MinAmount        int      `json:"min_amount"`
	DiscountType     string   `json:"discount_type"      binding:"required"`
	DiscountValue    int      `json:"discount_value"     binding:"required,min=1"`
}

// POST /api/discount-rules
func (h *Handler) CreateDiscountRule(c *gin.Context) {
	storeID, _ := c.Get("store_id")
	var req discountRuleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "入力が不正です"})
		return
	}
	if req.TargetProductIDs == nil {
		req.TargetProductIDs = []string{}
	}

	var rule models.DiscountRule
	if err := h.db.Get(&rule, `
		INSERT INTO discount_rules
			(store_id, name, is_active, condition_type, target_product_ids,
			 min_quantity, min_amount, discount_type, discount_value)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		RETURNING *`,
		storeID, req.Name, req.IsActive, req.ConditionType,
		pq.StringArray(req.TargetProductIDs),
		req.MinQuantity, req.MinAmount, req.DiscountType, req.DiscountValue,
	); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "作成失敗"})
		return
	}
	c.JSON(http.StatusCreated, rule)
}

// PATCH /api/discount-rules/:id
func (h *Handler) UpdateDiscountRule(c *gin.Context) {
	storeID, _ := c.Get("store_id")
	id := c.Param("id")
	var req discountRuleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "入力が不正です"})
		return
	}
	if req.TargetProductIDs == nil {
		req.TargetProductIDs = []string{}
	}

	var rule models.DiscountRule
	if err := h.db.Get(&rule, `
		UPDATE discount_rules
		SET name=$1, is_active=$2, condition_type=$3, target_product_ids=$4,
		    min_quantity=$5, min_amount=$6, discount_type=$7, discount_value=$8
		WHERE id=$9 AND store_id=$10
		RETURNING *`,
		req.Name, req.IsActive, req.ConditionType,
		pq.StringArray(req.TargetProductIDs),
		req.MinQuantity, req.MinAmount, req.DiscountType, req.DiscountValue,
		id, storeID,
	); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "ルールが見つかりません"})
		return
	}
	c.JSON(http.StatusOK, rule)
}

// DELETE /api/discount-rules/:id
func (h *Handler) DeleteDiscountRule(c *gin.Context) {
	storeID, _ := c.Get("store_id")
	id := c.Param("id")

	res, err := h.db.Exec(`
		DELETE FROM discount_rules
		WHERE id=$1 AND store_id=$2`, id, storeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "削除失敗"})
		return
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "ルールが見つかりません"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "削除しました"})
}

