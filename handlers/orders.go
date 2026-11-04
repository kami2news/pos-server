package handlers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kami2news/pos-server/models"
)

// GET /api/orders  → 当日注文一覧（items付き）
func (h *Handler) ListOrders(c *gin.Context) {
	storeID, _ := c.Get("store_id")

	// 現在の営業日
	var bdID string
	if err := h.db.Get(&bdID, `
		SELECT id FROM business_days WHERE store_id=$1 AND is_active=TRUE`, storeID); err != nil {
		c.JSON(http.StatusOK, []gin.H{})
		return
	}

	var orders []models.Order
	if err := h.db.Select(&orders, `
		SELECT * FROM orders
		WHERE business_day_id=$1
		ORDER BY ordered_at`, bdID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "取得失敗"})
		return
	}

	result := make([]gin.H, 0, len(orders))
	for _, o := range orders {
		var items []models.OrderItem
		h.db.Select(&items, `SELECT * FROM order_items WHERE order_id=$1`, o.ID)
		if items == nil {
			items = []models.OrderItem{}
		}
		result = append(result, orderToMap(o, items))
	}
	c.JSON(http.StatusOK, result)
}

// POST /api/orders  → 注文作成
type orderItemReq struct {
	ProductID   string `json:"product_id"`
	ProductName string `json:"product_name" binding:"required"`
	UnitPrice   int    `json:"unit_price"   binding:"required"`
	Quantity    int    `json:"quantity"     binding:"required,min=1"`
}

type createOrderReq struct {
	Items          []orderItemReq `json:"items"           binding:"required,min=1"`
	Subtotal       int            `json:"subtotal"        binding:"required"`
	DiscountAmount int            `json:"discount_amount"`
	TotalAmount    int            `json:"total_amount"    binding:"required"`
	Status         string         `json:"status"`          // "pending" or "completed"
	PaymentMethod  *string        `json:"payment_method"`
	ReceivedAmount *int           `json:"received_amount"`
}

func (h *Handler) CreateOrder(c *gin.Context) {
	storeID, _ := c.Get("store_id")
	var req createOrderReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "入力が不正です: " + err.Error()})
		return
	}

	// 営業日
	var bdID string
	if err := h.db.Get(&bdID, `
		SELECT id FROM business_days WHERE store_id=$1 AND is_active=TRUE`, storeID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "営業日を開始してください"})
		return
	}

	// 連番（当日注文数+1）
	var count int
	h.db.Get(&count, `SELECT COUNT(*) FROM orders WHERE business_day_id=$1`, bdID)
	orderNumber := fmt.Sprintf("%03d", count+1)

	status := "pending"
	if req.Status == "completed" || req.Status == "readyForPickup" {
		status = req.Status
	}

	tx := h.db.MustBegin()

	var order models.Order
	if err := tx.Get(&order, `
		INSERT INTO orders
		  (store_id, business_day_id, order_number, subtotal, discount_amount,
		   total_amount, status, payment_method, received_amount)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		RETURNING *`,
		storeID, bdID, orderNumber,
		req.Subtotal, req.DiscountAmount, req.TotalAmount,
		status, req.PaymentMethod, req.ReceivedAmount); err != nil {
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

// PATCH /api/orders/:id/complete  → 精算完了（即時）
func (h *Handler) CompleteOrder(c *gin.Context) {
	h.updateWithPayment(c, "completed")
}

// PATCH /api/orders/:id/ready  → 支払い済み・お渡し待ち
func (h *Handler) ReadyOrder(c *gin.Context) {
	h.updateWithPayment(c, "readyForPickup")
}

// PATCH /api/orders/:id/handover  → お渡し完了
func (h *Handler) HandoverOrder(c *gin.Context) {
	h.updateStatus(c, "completed")
}

// PATCH /api/orders/:id/cancel  → キャンセル
func (h *Handler) CancelOrder(c *gin.Context) {
	h.updateStatus(c, "cancelled")
}

// ── 内部ヘルパー ──────────────────────────────────────────────

type paymentReq struct {
	PaymentMethod  string `json:"payment_method" binding:"required,oneof=cash card"`
	ReceivedAmount *int   `json:"received_amount"`
}

func (h *Handler) updateWithPayment(c *gin.Context, newStatus string) {
	storeID, _ := c.Get("store_id")
	id := c.Param("id")

	var req paymentReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "支払い情報が必要です"})
		return
	}

	res, err := h.db.Exec(`
		UPDATE orders
		SET status=$1, payment_method=$2, received_amount=$3, updated_at=NOW()
		WHERE id=$4 AND store_id=$5`,
		newStatus, req.PaymentMethod, req.ReceivedAmount, id, storeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失敗"})
		return
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "注文が見つかりません"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": newStatus})
}

func (h *Handler) updateStatus(c *gin.Context, newStatus string) {
	storeID, _ := c.Get("store_id")
	id := c.Param("id")

	res, err := h.db.Exec(`
		UPDATE orders SET status=$1, updated_at=NOW()
		WHERE id=$2 AND store_id=$3`,
		newStatus, id, storeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失敗"})
		return
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "注文が見つかりません"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": newStatus})
}

func orderToMap(o models.Order, items []models.OrderItem) gin.H {
	return gin.H{
		"id":              o.ID,
		"order_number":    o.OrderNumber,
		"subtotal":        o.Subtotal,
		"discount_amount": o.DiscountAmount,
		"total_amount":    o.TotalAmount,
		"status":          o.Status,
		"payment_method":  o.PaymentMethod,
		"received_amount": o.ReceivedAmount,
		"ordered_at":      o.OrderedAt,
		"updated_at":      o.UpdatedAt,
		"items":           items,
	}
}
