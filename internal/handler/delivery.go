package handler

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/zachbroad/nitrohook/internal/model"
	"github.com/zachbroad/nitrohook/internal/store"
)

type DeliveryHandler struct {
	store *store.Store
	rdb   *redis.Client
}

func NewDeliveryHandler(s *store.Store, rdb *redis.Client) *DeliveryHandler {
	return &DeliveryHandler{store: s, rdb: rdb}
}

func (h *DeliveryHandler) List(c *gin.Context) {
	var sourceSlug *string
	if s := c.Query("source"); s != "" {
		sourceSlug = &s
	}

	limit := 50
	if l := c.Query("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}

	deliveries, err := h.store.Deliveries.List(c.Request.Context(), sourceSlug, limit)
	if err != nil {
		c.String(http.StatusInternalServerError, "failed to list deliveries")
		return
	}

	if deliveries == nil {
		c.Data(http.StatusOK, "application/json", []byte("[]"))
		return
	}
	c.JSON(http.StatusOK, deliveries)
}

func (h *DeliveryHandler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.String(http.StatusBadRequest, "invalid delivery id")
		return
	}

	delivery, err := h.store.Deliveries.GetByID(c.Request.Context(), id)
	if err != nil {
		c.String(http.StatusNotFound, "delivery not found")
		return
	}

	c.JSON(http.StatusOK, delivery)
}

func (h *DeliveryHandler) ListAttempts(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.String(http.StatusBadRequest, "invalid delivery id")
		return
	}

	// Verify delivery exists
	if _, err := h.store.Deliveries.GetByID(c.Request.Context(), id); err != nil {
		c.String(http.StatusNotFound, "delivery not found")
		return
	}

	attempts, err := h.store.Deliveries.ListAttemptsByDelivery(c.Request.Context(), id)
	if err != nil {
		c.String(http.StatusInternalServerError, "failed to list attempts")
		return
	}

	if attempts == nil {
		c.Data(http.StatusOK, "application/json", []byte("[]"))
		return
	}
	c.JSON(http.StatusOK, attempts)
}

func (h *DeliveryHandler) publishForward(ctx context.Context, id uuid.UUID) error {
	return h.rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: "deliveries", MaxLen: 10000, Approx: true,
		Values: map[string]any{"delivery_id": id.String(), "force": "1"},
	}).Err()
}

// Forward re-queues a single recorded delivery for fan-out.
func (h *DeliveryHandler) Forward(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.String(http.StatusBadRequest, "invalid delivery id")
		return
	}
	d, err := h.store.Deliveries.GetByID(c.Request.Context(), id)
	if err != nil {
		c.String(http.StatusNotFound, "delivery not found")
		return
	}
	if d.Status != model.DeliveryRecorded {
		c.JSON(http.StatusOK, gin.H{"status": "skipped"})
		return
	}
	if err := h.store.Deliveries.UpdateStatus(c.Request.Context(), id, model.DeliveryPending); err != nil {
		c.String(http.StatusInternalServerError, "failed to forward delivery")
		return
	}
	if err := h.publishForward(c.Request.Context(), id); err != nil {
		slog.Error("forward publish failed", "error", err, "delivery_id", id)
		_ = h.store.Deliveries.UpdateStatus(c.Request.Context(), id, model.DeliveryRecorded)
		c.String(http.StatusInternalServerError, "failed to forward delivery")
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "forwarded"})
}

// ForwardAll re-queues every recorded delivery for a source.
func (h *DeliveryHandler) ForwardAll(c *gin.Context) {
	slug := c.Param("sourceSlug")
	if _, err := h.store.Sources.GetBySlug(c.Request.Context(), slug); err != nil {
		c.String(http.StatusNotFound, "source not found")
		return
	}
	deliveries, err := h.store.Deliveries.List(c.Request.Context(), &slug, 200)
	if err != nil {
		c.String(http.StatusInternalServerError, "failed to list deliveries")
		return
	}
	forwarded := 0
	for _, d := range deliveries {
		if d.Status != model.DeliveryRecorded {
			continue
		}
		if err := h.store.Deliveries.UpdateStatus(c.Request.Context(), d.ID, model.DeliveryPending); err != nil {
			continue
		}
		if err := h.publishForward(c.Request.Context(), d.ID); err != nil {
			_ = h.store.Deliveries.UpdateStatus(c.Request.Context(), d.ID, model.DeliveryRecorded)
			continue
		}
		forwarded++
	}
	c.JSON(http.StatusOK, gin.H{"forwarded": forwarded})
}
