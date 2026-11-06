package handlers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kami2news/pos-server/models"
)

// ─────────────────────────────────────────────────────────────
// 注文者モード（認証不要）
// お客様がQRコードからアクセスして注文するための公開API
// ─────────────────────────────────────────────────────────────

// GET /public/:storeCode/info
// 店舗情報と商品一覧を返す（営業日が開いていない場合は is_open=false）
func (h *Handler) PublicGetStoreInfo(c *gin.Context) {
	storeCode := c.Param("storeCode")

	var store models.Store
	if err := h.db.Get(&store, `
		SELECT id, name, theme_color FROM stores WHERE name = $1`, storeCode); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "店舗が見つかりません"})
		return
	}

	// 営業日が開いているか確認
	var bdID string
	isOpen := true
	if err := h.db.Get(&bdID, `
		SELECT id FROM business_days WHERE store_id=$1 AND is_active=TRUE`, store.ID); err != nil {
		isOpen = false
	}

	var products []models.Product
	if isOpen {
		if err := h.db.Select(&products, `
			SELECT * FROM products
			WHERE store_id=$1 AND is_active=TRUE
			ORDER BY display_order, created_at`, store.ID); err != nil {
			products = []models.Product{}
		}
	}
	if products == nil {
		products = []models.Product{}
	}

	c.JSON(http.StatusOK, gin.H{
		"store_name":  store.Name,
		"theme_color": store.ThemeColor,
		"is_open":     isOpen,
		"products":    products,
	})
}

// POST /public/:storeCode/orders
// お客様からの注文作成（認証不要、現在の営業日に自動紐付け）
type publicOrderItemReq struct {
	ProductID   string `json:"product_id"`
	ProductName string `json:"product_name" binding:"required"`
	UnitPrice   int    `json:"unit_price"   binding:"required"`
	Quantity    int    `json:"quantity"     binding:"required,min=1"`
}

type publicCreateOrderReq struct {
	Items          []publicOrderItemReq `json:"items"           binding:"required,min=1"`
	Subtotal       int                  `json:"subtotal"        binding:"required"`
	DiscountAmount int                  `json:"discount_amount"`
	TotalAmount    int                  `json:"total_amount"    binding:"required"`
}

func (h *Handler) PublicCreateOrder(c *gin.Context) {
	storeCode := c.Param("storeCode")

	var store models.Store
	if err := h.db.Get(&store, `
		SELECT id FROM stores WHERE name = $1`, storeCode); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "店舗が見つかりません"})
		return
	}

	var req publicCreateOrderReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "入力が不正です"})
		return
	}

	// 現在の営業日を取得
	var bdID string
	if err := h.db.Get(&bdID, `
		SELECT id FROM business_days WHERE store_id=$1 AND is_active=TRUE`, store.ID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "現在受け付けておりません"})
		return
	}

	// トランザクション内で連番発番（重複防止）
	tx := h.db.MustBegin()

	var count int
	if err := tx.Get(&count, `
		SELECT COUNT(*) FROM orders WHERE business_day_id=$1`, bdID); err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "番号発番失敗"})
		return
	}
	orderNumber := fmt.Sprintf("%03d", count+1)

	var order models.Order
	if err := tx.Get(&order, `
		INSERT INTO orders
		  (store_id, business_day_id, order_number, subtotal, discount_amount,
		   total_amount, status)
		VALUES ($1,$2,$3,$4,$5,$6,'pending')
		RETURNING *`,
		store.ID, bdID, orderNumber,
		req.Subtotal, req.DiscountAmount, req.TotalAmount); err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "注文作成失敗"})
		return
	}

	var items []models.OrderItem
	for _, item := range req.Items {
		var oi models.OrderItem
		if err := tx.Get(&oi, `
			INSERT INTO order_items
			  (order_id, product_id, product_name, unit_price, quantity)
			VALUES ($1,$2,$3,$4,$5) RETURNING *`,
			order.ID, nilableString(item.ProductID),
			item.ProductName, item.UnitPrice, item.Quantity); err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "注文明細作成失敗"})
			return
		}
		items = append(items, oi)
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "コミット失敗"})
		return
	}

	if items == nil {
		items = []models.OrderItem{}
	}
	c.JSON(http.StatusCreated, orderToMap(order, items))
}

// GET /public/orders/:orderId
// 注文ステータス確認（認証不要・お客様が整理番号ページでポーリング）
func (h *Handler) PublicGetOrder(c *gin.Context) {
	id := c.Param("orderId")

	var order models.Order
	if err := h.db.Get(&order, `SELECT * FROM orders WHERE id=$1`, id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "注文が見つかりません"})
		return
	}

	var items []models.OrderItem
	h.db.Select(&items, `SELECT * FROM order_items WHERE order_id=$1`, id)
	if items == nil {
		items = []models.OrderItem{}
	}

	c.JSON(http.StatusOK, orderToMap(order, items))
}
