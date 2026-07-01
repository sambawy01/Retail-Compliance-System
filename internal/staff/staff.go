// Package staff provides staff performance, attendance, and holiday management.
// It aggregates detection data linked to persons via the payload to compute
// performance scores and generates AI-style reports algorithmically.
package staff

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/sambawy01/Retail-Compliance-System/pkg/database"
)

// StaffProfile is the full staff member profile with aggregated stats.
type StaffProfile struct {
	PersonID      string     `json:"person_id"`
	DisplayName   string     `json:"display_name"`
	Kind          string     `json:"kind"`
	Phone         string     `json:"phone,omitempty"`
	JobRole       string     `json:"job_role,omitempty"`
	Department    string     `json:"department,omitempty"`
	PhotoURL      string     `json:"photo_url,omitempty"`
	EmployeeCode  string     `json:"employee_id_code,omitempty"`
	ShiftStart    string     `json:"shift_start,omitempty"`
	ShiftEnd      string     `json:"shift_end,omitempty"`
	HireDate      *time.Time `json:"hire_date,omitempty"`
	Status        string     `json:"status,omitempty"`
	EnrolledAt    time.Time  `json:"enrolled_at"`

	// Aggregated stats
	PerformanceScore float64       `json:"performance_score"`
	TotalEvents      int           `json:"total_events"`
	CriticalEvents   int           `json:"critical_events"`
	WarningEvents    int           `json:"warning_events"`
	InfoEvents       int           `json:"info_events"`
	EventBreakdown   []EventCount  `json:"event_breakdown,omitempty"`
	Attendance       AttendanceSummary `json:"attendance"`
	Holidays         []Holiday     `json:"holidays,omitempty"`
}

type EventCount struct {
	EventType string `json:"event_type"`
	Count     int    `json:"count"`
	Severity  string `json:"severity"`
}

type AttendanceSummary struct {
	PresentDays   int     `json:"present_days"`
	LateDays      int     `json:"late_days"`
	AbsentDays    int     `json:"absent_days"`
	HalfDays      int     `json:"half_days"`
	RemoteDays    int     `json:"remote_days"`
	TotalDays     int     `json:"total_days"`
	LateMinutes   int     `json:"late_minutes"`
	OvertimeMins  int     `json:"overtime_minutes"`
	AttendanceRate float64 `json:"attendance_rate"`
}

type Holiday struct {
	HolidayID  string     `json:"holiday_id"`
	StartDate  time.Time  `json:"start_date"`
	EndDate    time.Time  `json:"end_date"`
	Type       string     `json:"type"`
	Status     string     `json:"status"`
	Reason     string     `json:"reason,omitempty"`
}

type AttendanceRecord struct {
	AttendanceID   string     `json:"attendance_id"`
	Date           time.Time  `json:"date"`
	CheckIn        *time.Time `json:"check_in,omitempty"`
	CheckOut       *time.Time `json:"check_out,omitempty"`
	Status         string     `json:"status"`
	LateMinutes    int        `json:"late_minutes"`
	OvertimeMinutes int       `json:"overtime_minutes"`
	Notes          string     `json:"notes,omitempty"`
}

// Service provides staff management operations.
type Service struct {
	pool *database.Pool
}

// New creates a staff Service.
func New(pool *database.Pool) *Service {
	return &Service{pool: pool}
}

// ListStaff returns all enrolled staff with aggregated performance stats.
func (s *Service) ListStaff(ctx context.Context) ([]StaffProfile, error) {
	var out []StaffProfile
	err := database.TenantTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT person_id, display_name, kind, phone, job_role, department,
			       photo_url, employee_id_code, shift_start, shift_end, hire_date,
			       status, enrolled_at
			FROM identity_persons
			WHERE revoked_at IS NULL
			ORDER BY display_name`)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var p StaffProfile
			if err := rows.Scan(
				&p.PersonID, &p.DisplayName, &p.Kind, &p.Phone, &p.JobRole, &p.Department,
				&p.PhotoURL, &p.EmployeeCode, &p.ShiftStart, &p.ShiftEnd, &p.HireDate,
				&p.Status, &p.EnrolledAt,
			); err != nil {
				return err
			}
			out = append(out, p)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, fmt.Errorf("staff: list: %w", err)
	}

	// Aggregate stats for each person
	for i := range out {
		s.aggregateStats(ctx, &out[i])
		s.aggregateAttendance(ctx, &out[i])
		s.loadHolidays(ctx, &out[i])
	}

	return out, nil
}

// GetStaff returns a single staff member with full profile and stats.
func (s *Service) GetStaff(ctx context.Context, personID string) (StaffProfile, error) {
	var p StaffProfile
	err := database.TenantTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
			SELECT person_id, display_name, kind, phone, job_role, department,
			       photo_url, employee_id_code, shift_start, shift_end, hire_date,
			       status, enrolled_at
			FROM identity_persons
			WHERE person_id = $1 AND revoked_at IS NULL`,
			personID,
		).Scan(
			&p.PersonID, &p.DisplayName, &p.Kind, &p.Phone, &p.JobRole, &p.Department,
			&p.PhotoURL, &p.EmployeeCode, &p.ShiftStart, &p.ShiftEnd, &p.HireDate,
			&p.Status, &p.EnrolledAt,
		)
	})
	if err != nil {
		return StaffProfile{}, fmt.Errorf("staff: get: %w", err)
	}

	s.aggregateStats(ctx, &p)
	s.aggregateAttendance(ctx, &p)
	s.loadHolidays(ctx, &p)

	return p, nil
}

// aggregateStats counts detections linked to this person via payload->person_id
// and computes a performance score.
func (s *Service) aggregateStats(ctx context.Context, p *StaffProfile) {
	err := database.TenantTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		// Count detections by severity where payload contains this person_id
		rows, err := tx.Query(ctx, `
			SELECT severity, COUNT(*)
			FROM vision_detections
			WHERE payload->>'person_id' = $1
			   OR payload->>'matched_person' = $1
			GROUP BY severity`, p.PersonID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var sev string
			var count int
			if err := rows.Scan(&sev, &count); err != nil {
				return err
			}
			switch sev {
			case "critical":
				p.CriticalEvents = count
			case "warning":
				p.WarningEvents = count
			case "info":
				p.InfoEvents = count
			}
			p.TotalEvents += count
		}

		// Event breakdown by type
		rows2, err := tx.Query(ctx, `
			SELECT event_type, severity, COUNT(*) as cnt
			FROM vision_detections
			WHERE payload->>'person_id' = $1
			   OR payload->>'matched_person' = $1
			GROUP BY event_type, severity
			ORDER BY cnt DESC
			LIMIT 10`, p.PersonID)
		if err != nil {
			return err
		}
		defer rows2.Close()
		for rows2.Next() {
			var ec EventCount
			if err := rows2.Scan(&ec.EventType, &ec.Severity, &ec.Count); err != nil {
				return err
			}
			p.EventBreakdown = append(p.EventBreakdown, ec)
		}

		return nil
	})
	if err != nil {
		return
	}

	// Compute performance score:
	// Start at 100, deduct: critical=-15, warning=-5, info=-1
	// Floor at 0, cap at 100
	score := 100.0
	score -= float64(p.CriticalEvents) * 15
	score -= float64(p.WarningEvents) * 5
	score -= float64(p.InfoEvents) * 1
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}
	p.PerformanceScore = score
}

// aggregateAttendance loads attendance summary for the last 30 days.
func (s *Service) aggregateAttendance(ctx context.Context, p *StaffProfile) {
	err := database.TenantTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		row := tx.QueryRow(ctx, `
			SELECT
				COUNT(*) FILTER (WHERE status = 'present') as present,
				COUNT(*) FILTER (WHERE status = 'late') as late,
				COUNT(*) FILTER (WHERE status = 'absent') as absent,
				COUNT(*) FILTER (WHERE status = 'half_day') as half,
				COUNT(*) FILTER (WHERE status = 'remote') as remote,
				COUNT(*) as total,
				COALESCE(SUM(late_minutes), 0) as late_mins,
				COALESCE(SUM(overtime_minutes), 0) as ot_mins
			FROM staff_attendance
			WHERE person_id = $1 AND date >= NOW() - INTERVAL '30 days'`, p.PersonID)

		var present, late, absent, half, remote, total, lateMins, otMins int
		if err := row.Scan(&present, &late, &absent, &half, &remote, &total, &lateMins, &otMins); err != nil {
			return err
		}
		p.Attendance = AttendanceSummary{
			PresentDays:    present,
			LateDays:       late,
			AbsentDays:     absent,
			HalfDays:       half,
			RemoteDays:     remote,
			TotalDays:      total,
			LateMinutes:    lateMins,
			OvertimeMins:   otMins,
		}
		if total > 0 {
			p.Attendance.AttendanceRate = float64(present+remote+half) / float64(total) * 100
		} else {
			p.Attendance.AttendanceRate = 100
		}
		return nil
	})
	if err != nil {
		return
	}
}

// loadHolidays loads holiday/leave records for this person.
func (s *Service) loadHolidays(ctx context.Context, p *StaffProfile) {
	err := database.TenantTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT holiday_id, start_date, end_date, type, status, reason
			FROM staff_holidays
			WHERE person_id = $1
			ORDER BY start_date DESC LIMIT 20`, p.PersonID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var h Holiday
			if err := rows.Scan(&h.HolidayID, &h.StartDate, &h.EndDate, &h.Type, &h.Status, &h.Reason); err != nil {
				return err
			}
			p.Holidays = append(p.Holidays, h)
		}
		return nil
	})
	if err != nil {
		return
	}
}

// GenerateReport produces an AI-style text report for a staff member.
func (s *Service) GenerateReport(ctx context.Context, personID string) (string, error) {
	p, err := s.GetStaff(ctx, personID)
	if err != nil {
		return "", err
	}

	var report strings.Builder
	report.WriteString(fmt.Sprintf("STAFF PERFORMANCE REPORT: %s\n", p.DisplayName))
	report.WriteString(fmt.Sprintf("Generated: %s\n\n", time.Now().Format("January 2, 2006 at 15:04")))

	// Profile
	report.WriteString("PROFILE\n")
	report.WriteString(fmt.Sprintf("  Role: %s\n", p.JobRole))
	if p.Department != "" {
		report.WriteString(fmt.Sprintf("  Department: %s\n", p.Department))
	}
	report.WriteString(fmt.Sprintf("  Status: %s\n", p.Status))
	if p.ShiftStart != "" {
		report.WriteString(fmt.Sprintf("  Shift: %s - %s\n", p.ShiftStart, p.ShiftEnd))
	}
	report.WriteString("\n")

	// Performance Score
	scoreLabel := "Excellent"
	if p.PerformanceScore < 80 {
		scoreLabel = "Good"
	}
	if p.PerformanceScore < 60 {
		scoreLabel = "Needs Improvement"
	}
	if p.PerformanceScore < 40 {
		scoreLabel = "Poor"
	}
	report.WriteString(fmt.Sprintf("PERFORMANCE SCORE: %.0f%% (%s)\n", p.PerformanceScore, scoreLabel))
	report.WriteString(fmt.Sprintf("  Total compliance events: %d\n", p.TotalEvents))
	report.WriteString(fmt.Sprintf("  Critical: %d | Warning: %d | Info: %d\n\n", p.CriticalEvents, p.WarningEvents, p.InfoEvents))

	// Event breakdown
	if len(p.EventBreakdown) > 0 {
		report.WriteString("TOP VIOLATIONS\n")
		for _, ec := range p.EventBreakdown {
			report.WriteString(fmt.Sprintf("  %s (%s): %d occurrences\n", ec.EventType, ec.Severity, ec.Count))
		}
		report.WriteString("\n")
	}

	// Attendance
	report.WriteString("ATTENDANCE (Last 30 Days)\n")
	report.WriteString(fmt.Sprintf("  Attendance rate: %.1f%%\n", p.Attendance.AttendanceRate))
	report.WriteString(fmt.Sprintf("  Present: %d | Late: %d | Absent: %d | Half-day: %d | Remote: %d\n",
		p.Attendance.PresentDays, p.Attendance.LateDays, p.Attendance.AbsentDays, p.Attendance.HalfDays, p.Attendance.RemoteDays))
	if p.Attendance.LateMinutes > 0 {
		report.WriteString(fmt.Sprintf("  Total late minutes: %d\n", p.Attendance.LateMinutes))
	}
	if p.Attendance.OvertimeMins > 0 {
		report.WriteString(fmt.Sprintf("  Overtime minutes: %d\n", p.Attendance.OvertimeMins))
	}
	report.WriteString("\n")

	// Holidays
	if len(p.Holidays) > 0 {
		report.WriteString("LEAVE & HOLIDAYS\n")
		for _, h := range p.Holidays {
			days := int(h.EndDate.Sub(h.StartDate).Hours()/24) + 1
			report.WriteString(fmt.Sprintf("  %s to %s (%d days) - %s [%s]\n",
				h.StartDate.Format("Jan 2"), h.EndDate.Format("Jan 2, 2006"), days, h.Type, h.Status))
		}
		report.WriteString("\n")
	}

	// AI Assessment
	report.WriteString("AI ASSESSMENT\n")
	if p.PerformanceScore >= 80 {
		report.WriteString("  This employee demonstrates strong compliance with store policies. ")
		if p.Attendance.LateDays > 0 {
			report.WriteString(fmt.Sprintf("However, %d late arrivals in the past 30 days should be addressed. ", p.Attendance.LateDays))
		}
		report.WriteString("Continue to monitor and recognize good performance.\n")
	} else if p.PerformanceScore >= 60 {
		report.WriteString("  This employee shows acceptable performance but has room for improvement. ")
		if p.CriticalEvents > 0 {
			report.WriteString(fmt.Sprintf("The %d critical violations require immediate coaching. ", p.CriticalEvents))
		}
		report.WriteString("A performance review meeting is recommended.\n")
	} else {
		report.WriteString("  This employee's performance is below acceptable thresholds. ")
		report.WriteString(fmt.Sprintf("With %d critical and %d warning violations, a formal improvement plan is required. ", p.CriticalEvents, p.WarningEvents))
		if p.Attendance.AbsentDays > 3 {
			report.WriteString("Frequent absences are also a concern. ")
		}
		report.WriteString("Consider escalation to management.\n")
	}

	return report.String(), nil
}