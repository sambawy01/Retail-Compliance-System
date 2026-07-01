// Package api — staff_handlers.go contains staff performance and management handlers.
package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/sambawy01/Retail-Compliance-System/pkg/database"
)

func (s *Server) listStaff(w http.ResponseWriter, r *http.Request) {
	profiles, err := s.staff.ListStaff(r.Context())
	if err != nil {
		slog.Error("list_staff_failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to list staff")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"staff": profiles})
}

func (s *Server) getStaff(w http.ResponseWriter, r *http.Request) {
	personID := chi.URLParam(r, "personID")
	profile, err := s.staff.GetStaff(r.Context(), personID)
	if err != nil {
		writeError(w, http.StatusNotFound, "staff member not found")
		return
	}
	writeJSON(w, http.StatusOK, profile)
}

func (s *Server) getStaffReport(w http.ResponseWriter, r *http.Request) {
	personID := chi.URLParam(r, "personID")
	report, err := s.staff.GenerateReport(r.Context(), personID)
	if err != nil {
		slog.Error("staff_report_failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to generate report")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"report": report})
}

func (s *Server) getStaffAttendance(w http.ResponseWriter, r *http.Request) {
	personID := chi.URLParam(r, "personID")
	var records []map[string]any
	err := database.TenantTx(r.Context(), s.pool, func(ctx context.Context, tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT attendance_id, date, check_in, check_out, status, late_minutes, overtime_minutes, notes
			FROM staff_attendance WHERE person_id = $1 ORDER BY date DESC LIMIT 90`, personID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var id, status, notes string
			var date time.Time
			var checkIn, checkOut *time.Time
			var lateMins, otMins int
			if err := rows.Scan(&id, &date, &checkIn, &checkOut, &status, &lateMins, &otMins, &notes); err != nil {
				return err
			}
			records = append(records, map[string]any{
				"attendance_id": id, "date": date, "check_in": checkIn, "check_out": checkOut,
				"status": status, "late_minutes": lateMins, "overtime_minutes": otMins, "notes": notes,
			})
		}
		return rows.Err()
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get attendance")
		return
	}
	if records == nil {
		records = []map[string]any{}
	}
	writeJSON(w, http.StatusOK, records)
}

func (s *Server) recordAttendance(w http.ResponseWriter, r *http.Request) {
	personID := chi.URLParam(r, "personID")
	var in struct {
		Date            string `json:"date"`
		CheckIn         string `json:"check_in"`
		CheckOut        string `json:"check_out"`
		Status          string `json:"status"`
		LateMinutes     int    `json:"late_minutes"`
		OvertimeMinutes int    `json:"overtime_minutes"`
		Notes           string `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	err := database.TenantTx(r.Context(), s.pool, func(ctx context.Context, tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			INSERT INTO staff_attendance (org_id, person_id, date, check_in, check_out, status, late_minutes, overtime_minutes, notes)
			VALUES (current_setting('app.current_org_id', true)::uuid, $1, $2, $3, $4, $5, $6, $7, $8)
			ON CONFLICT (person_id, date) DO UPDATE SET check_in = EXCLUDED.check_in, check_out = EXCLUDED.check_out,
				status = EXCLUDED.status, late_minutes = EXCLUDED.late_minutes, overtime_minutes = EXCLUDED.overtime_minutes, notes = EXCLUDED.notes`,
			personID, in.Date, in.CheckIn, in.CheckOut, in.Status, in.LateMinutes, in.OvertimeMinutes, in.Notes)
		return err
	})
	if err != nil {
		slog.Error("record_attendance_failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to record attendance")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"status": "recorded"})
}

func (s *Server) getStaffHolidays(w http.ResponseWriter, r *http.Request) {
	personID := chi.URLParam(r, "personID")
	var records []map[string]any
	err := database.TenantTx(r.Context(), s.pool, func(ctx context.Context, tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT holiday_id, start_date, end_date, type, status, reason
			FROM staff_holidays WHERE person_id = $1 ORDER BY start_date DESC`, personID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var id, htype, status, reason string
			var start, end time.Time
			if err := rows.Scan(&id, &start, &end, &htype, &status, &reason); err != nil {
				return err
			}
			records = append(records, map[string]any{
				"holiday_id": id, "start_date": start, "end_date": end,
				"type": htype, "status": status, "reason": reason,
			})
		}
		return rows.Err()
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get holidays")
		return
	}
	if records == nil {
		records = []map[string]any{}
	}
	writeJSON(w, http.StatusOK, records)
}

func (s *Server) requestHoliday(w http.ResponseWriter, r *http.Request) {
	personID := chi.URLParam(r, "personID")
	var in struct {
		StartDate string `json:"start_date"`
		EndDate   string `json:"end_date"`
		Type      string `json:"type"`
		Reason    string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	err := database.TenantTx(r.Context(), s.pool, func(ctx context.Context, tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			INSERT INTO staff_holidays (org_id, person_id, start_date, end_date, type, reason)
			VALUES (current_setting('app.current_org_id', true)::uuid, $1, $2, $3, $4, $5)`,
			personID, in.StartDate, in.EndDate, in.Type, in.Reason)
		return err
	})
	if err != nil {
		slog.Error("request_holiday_failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to request holiday")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"status": "requested"})
}