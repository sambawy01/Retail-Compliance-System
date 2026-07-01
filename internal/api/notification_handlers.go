// Package api — notification_handlers.go contains notification rule HTTP handlers.
package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/sambawy01/Retail-Compliance-System/internal/notifications"
)

func (s *Server) listNotificationRules(w http.ResponseWriter, r *http.Request) {
	rules, err := s.notifications.ListRules(r.Context())
	if err != nil {
		slog.Error("list_notification_rules_failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to list notification rules")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"rules": rules})
}

func (s *Server) createNotificationRule(w http.ResponseWriter, r *http.Request) {
	var in notifications.CreateRuleInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if in.EventType == "" || in.Severity == "" || in.Channel == "" || in.Target == "" {
		writeError(w, http.StatusBadRequest, "event_type, severity, channel, and target are required")
		return
	}
	rule, err := s.notifications.CreateRule(r.Context(), in)
	if err != nil {
		slog.Error("create_notification_rule_failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to create notification rule")
		return
	}
	writeJSON(w, http.StatusCreated, rule)
}

func (s *Server) updateNotificationRule(w http.ResponseWriter, r *http.Request) {
	ruleID := chi.URLParam(r, "ruleID")
	var in notifications.UpdateRuleInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	rule, err := s.notifications.UpdateRule(r.Context(), ruleID, in)
	if err != nil {
		if errors.Is(err, notifications.ErrRuleNotFound) {
			writeError(w, http.StatusNotFound, "notification rule not found")
			return
		}
		slog.Error("update_notification_rule_failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to update notification rule")
		return
	}
	writeJSON(w, http.StatusOK, rule)
}

func (s *Server) deleteNotificationRule(w http.ResponseWriter, r *http.Request) {
	ruleID := chi.URLParam(r, "ruleID")
	if err := s.notifications.DeleteRule(r.Context(), ruleID); err != nil {
		if errors.Is(err, notifications.ErrRuleNotFound) {
			writeError(w, http.StatusNotFound, "notification rule not found")
			return
		}
		slog.Error("delete_notification_rule_failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to delete notification rule")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}