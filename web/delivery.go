package web

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (h *Handler) Deliveries(c *gin.Context) {
	sources, err := h.store.Sources.List(c.Request.Context())
	if err != nil {
		slog.Error("failed to list sources", "error", err)
		c.String(http.StatusInternalServerError, "Internal server error")
		return
	}
	sourceFilter := c.Query("source")
	var sourceSlug *string
	if sourceFilter != "" {
		sourceSlug = &sourceFilter
	}
	deliveries, err := h.store.Deliveries.List(c.Request.Context(), sourceSlug, 50)
	if err != nil {
		slog.Error("failed to list deliveries", "error", err)
		c.String(http.StatusInternalServerError, "Internal server error")
		return
	}
	h.render(c, "deliveries", deliveriesData{
		Nav:          "deliveries",
		Sources:      sources,
		Deliveries:   deliveries,
		SourceFilter: sourceFilter,
	})
}

func (h *Handler) DeliveryDetail(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid delivery ID")
		return
	}
	delivery, err := h.store.Deliveries.GetByID(c.Request.Context(), id)
	if err != nil {
		c.String(http.StatusNotFound, "Delivery not found")
		return
	}
	attempts, err := h.store.Deliveries.ListAttemptsByDelivery(c.Request.Context(), id)
	if err != nil {
		slog.Error("failed to list attempts", "error", err)
		c.String(http.StatusInternalServerError, "Internal server error")
		return
	}
	h.render(c, "delivery", deliveryData{
		Nav:      "deliveries",
		Delivery: delivery,
		Attempts: attempts,
	})
}
