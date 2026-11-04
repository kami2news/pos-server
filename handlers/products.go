package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kami2news/pos-server/models"
)

// GET /api/products
func (h *Handler) ListProducts(c *gin.Context) {
	storeID, _ := c.Get("store_id")
	var products []models.Product
	if err := h.db.Select(&products, `
		SELECT * FROM products
		WHERE store_id=$1 AND is_active=TRUE
		ORDER BY display_order, created_at`, storeID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "取得失敗"})
		return
	}
	if products == nil {
		products = []models.Product{}
	}
	c.JSON(http.StatusOK, products)
}

type productReq struct {
	Name         string `json:"name"          binding:"required"`
	Price        int    `json:"price"         binding:"required,min=0"`
	DisplayOrder int    `json:"display_order"`
}

// POST /api/products
func (h *Handler) CreateProduct(c *gin.Context) {
	storeID, _ := c.Get("store_id")
	var req productReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "入力が不正です"})
		return
	}

	var p models.Product
	if err := h.db.Get(&p, `
		INSERT INTO products (store_id, name, price, display_order)
		VALUES ($1, $2, $3, $4) RETURNING *`,
		storeID, req.Name, req.Price, req.DisplayOrder); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "作成失敗"})
		return
	}
	c.JSON(http.StatusCreated, p)
}

// PATCH /api/products/:id
func (h *Handler) UpdateProduct(c *gin.Context) {
	storeID, _ := c.Get("store_id")
	id := c.Param("id")
	var req productReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "入力が不正です"})
		return
	}

	var p models.Product
	if err := h.db.Get(&p, `
		UPDATE products SET name=$1, price=$2, display_order=$3
		WHERE id=$4 AND store_id=$5
		RETURNING *`,
		req.Name, req.Price, req.DisplayOrder, id, storeID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "商品が見つかりません"})
		return
	}
	c.JSON(http.StatusOK, p)
}

// DELETE /api/products/:id  （論理削除）
func (h *Handler) DeleteProduct(c *gin.Context) {
	storeID, _ := c.Get("store_id")
	id := c.Param("id")

	res, err := h.db.Exec(`
		UPDATE products SET is_active=FALSE
		WHERE id=$1 AND store_id=$2`, id, storeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "削除失敗"})
		return
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "商品が見つかりません"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "削除しました"})
}
