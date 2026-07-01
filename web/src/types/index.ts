// Type definitions for the Watch Dog frontend.
// These types mirror the Go backend structs and provide type safety
// across the API boundary.

// --- Auth ---
export interface User {
  user_id: string
  email: string
  display_name: string
  role: 'owner' | 'admin' | 'manager' | 'staff' | 'read_only'
  org_id: string
}

export interface LoginResponse {
  token: string
  user: User
}

// --- Vision ---
export interface Camera {
  camera_id: string
  org_id: string
  location_id?: string
  name: string
  rtsp_url: string
  status: 'online' | 'offline' | 'degraded' | 'pending'
  last_heartbeat_at?: string
  created_at: string
  updated_at: string
}

export interface Zone {
  zone_id: string
  org_id: string
  camera_id: string
  name: string
  kind: string
  polygon?: unknown
  capacity?: number
  created_at: string
}

export interface Detection {
  detection_id: string
  org_id: string
  location_id?: string
  camera_id?: string
  zone_id?: string
  event_type: string
  severity: 'critical' | 'warning' | 'info'
  confidence?: number
  payload?: Record<string, unknown>
  clip_id?: string
  detected_at: string
  created_at: string
}

export interface Clip {
  clip_id: string
  org_id: string
  location_id?: string
  camera_id?: string
  s3_bucket?: string
  s3_key?: string
  duration_seconds?: number
  starts_at?: string
  retention_tier: 'hot' | 'warm' | 'deleted'
  created_at: string
}

// --- Identity ---
export interface Person {
  person_id: string
  org_id: string
  kind: 'employee' | 'customer'
  display_name: string
  phone?: string
  job_role?: string
  photo_url?: string
  enrolled_at: string
  revoked_at?: string
  created_at: string
  updated_at: string
}

export interface MatchResult {
  person_id: string
  display_name: string
  similarity: number
  matched: boolean
}

// --- Notifications ---
export interface NotificationRule {
  rule_id: string
  org_id: string
  event_type: string
  severity: 'critical' | 'warning' | 'info'
  channel: 'telegram' | 'email' | 'sms' | 'dashboard'
  target: string
  enabled: boolean
  created_at: string
  updated_at: string
}

// --- Staff ---
export interface StaffProfile {
  person_id: string
  display_name: string
  kind: string
  phone?: string
  job_role?: string
  department?: string
  photo_url?: string
  employee_id_code?: string
  shift_start?: string
  shift_end?: string
  hire_date?: string
  status?: 'active' | 'inactive' | 'on_leave' | 'terminated'
  enrolled_at: string
  performance_score: number
  total_events: number
  critical_events: number
  warning_events: number
  info_events: number
  event_breakdown?: EventCount[]
  attendance: AttendanceSummary
  holidays?: Holiday[]
}

export interface EventCount {
  event_type: string
  count: number
  severity: string
}

export interface AttendanceSummary {
  present_days: number
  late_days: number
  absent_days: number
  half_days: number
  remote_days: number
  total_days: number
  late_minutes: number
  overtime_mins: number
  attendance_rate: number
}

export interface Holiday {
  holiday_id: string
  start_date: string
  end_date: string
  type: 'annual' | 'sick' | 'personal' | 'unpaid' | 'public_holiday'
  status: 'pending' | 'approved' | 'rejected'
  reason?: string
}

// --- API Response shapes ---
export interface ListResponse<T> {
  [key: string]: T[] | undefined
}

export interface ErrorResponse {
  error: string
}