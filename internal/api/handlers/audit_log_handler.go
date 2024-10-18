package handlers

import (
	"landmark-api/internal/services"
	"net/http"
	"strconv"
)

type AuditLogHandler struct {
	auditLogService services.AuditLogService
}

func NewAuditLogHandler(auditLogService services.AuditLogService) *AuditLogHandler {
	return &AuditLogHandler{
		auditLogService: auditLogService,
	}
}

func (h *AuditLogHandler) ListAuditLogs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("pageSize"))
	if pageSize < 1 {
		pageSize = 20
	}

	logs, total, err := h.auditLogService.GetAuditLogs(ctx, page, pageSize)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error fetching audit logs")
		return
	}

	response := map[string]interface{}{
		"logs":  logs,
		"total": total,
		"page":  page,
	}

	respondWithJSON(w, http.StatusOK, response)
}
