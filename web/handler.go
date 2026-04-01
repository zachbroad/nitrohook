package web

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"log/slog"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/zachbroad/nitrohook/internal/model"
	"github.com/zachbroad/nitrohook/internal/script"
	"github.com/zachbroad/nitrohook/internal/store"
)

var funcMap = template.FuncMap{
	"shortID": func(id uuid.UUID) string {
		s := id.String()
		if len(s) >= 8 {
			return s[:8]
		}
		return s
	},
	"formatTime": func(t time.Time) string {
		return t.Format("Jan 2, 2006 3:04 PM")
	},
	"formatJSON": func(data json.RawMessage) template.HTML {
		if data == nil {
			return "-"
		}
		var out bytes.Buffer
		if err := json.Indent(&out, data, "", "  "); err != nil {
			return template.HTML(template.HTMLEscapeString(string(data)))
		}
		return template.HTML(template.HTMLEscapeString(out.String()))
	},
	"derefInt": func(p *int) string {
		if p == nil {
			return "-"
		}
		return strconv.Itoa(*p)
	},
	"derefStr": func(p *string) string {
		if p == nil {
			return "-"
		}
		return *p
	},
	"marshalJSON": func(v any) template.HTML {
		b, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			return template.HTML(template.HTMLEscapeString(fmt.Sprintf("%v", v)))
		}
		return template.HTML(template.HTMLEscapeString(string(b)))
	},
	"configGet": func(cfg json.RawMessage, key string) string {
		if cfg == nil {
			return ""
		}
		var m map[string]any
		if err := json.Unmarshal(cfg, &m); err != nil {
			return ""
		}
		if v, ok := m[key]; ok {
			return fmt.Sprintf("%v", v)
		}
		return ""
	},
	"truncateJSON": func(data json.RawMessage, maxLen int) string {
		if data == nil {
			return "(empty)"
		}
		var out bytes.Buffer
		if err := json.Compact(&out, data); err != nil {
			s := string(data)
			if len(s) > maxLen {
				return s[:maxLen] + "…"
			}
			return s
		}
		s := out.String()
		if len(s) > maxLen {
			return s[:maxLen] + "…"
		}
		return s
	},
}

type Handler struct {
	store     *store.Store
	rdb       *redis.Client
	templates map[string]*template.Template
}

func NewHandler(s *store.Store, rdb *redis.Client) *Handler {
	h := &Handler{
		store:     s,
		rdb:       rdb,
		templates: make(map[string]*template.Template),
	}
	for _, page := range []string{"sources", "source-overview", "source-script", "source-actions", "source-events", "deliveries", "delivery"} {
		h.templates[page] = template.Must(
			template.New("").Funcs(funcMap).ParseFS(templateFS,
				"templates/layout.html",
				"templates/"+page+".html",
			),
		)
	}
	return h
}

func (h *Handler) render(c *gin.Context, page string, data any) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	if err := h.templates[page].ExecuteTemplate(c.Writer, "layout", data); err != nil {
		slog.Error("template render error", "page", page, "error", err)
	}
}

func (h *Handler) renderFragment(c *gin.Context, page string, fragment string, data any) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	if err := h.templates[page].ExecuteTemplate(c.Writer, fragment, data); err != nil {
		slog.Error("template render error", "page", page, "fragment", fragment, "error", err)
	}
}

// Page data types

type sourcesData struct {
	Nav          string
	Sources      []model.Source
	ActionCounts map[uuid.UUID]int
	Error        string
}

type sourceData struct {
	Nav           string
	Source        *model.Source
	Actions       []model.Action
	Deliveries    []model.Delivery
	WebhookURL    string
	ActiveTab     string
	Error         string
	ScriptError   string
	ScriptSuccess string
	EditAction    *model.Action
	ActionError   string
	ActionSuccess string
}

type scriptTestData struct {
	Result *script.TransformResult
	Error  string
}

type deliveriesData struct {
	Nav          string
	Sources      []model.Source
	Deliveries   []model.Delivery
	SourceFilter string
}

type deliveryData struct {
	Nav      string
	Delivery *model.Delivery
	Attempts []model.DeliveryAttempt
}
