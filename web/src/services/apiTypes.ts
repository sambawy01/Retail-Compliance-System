// Typed API client for the Watch Dog frontend.
// Provides type-safe wrappers around the axios instance.
import type {
  Camera, Zone, Detection, Clip, Person, MatchResult,
  NotificationRule, StaffProfile, User, LoginResponse,
} from '../types'

export interface CreateCameraInput {
  name: string
  rtsp_url: string
  location_id?: string
}

export interface EnrollPersonInput {
  display_name: string
  kind: 'employee' | 'customer'
  phone?: string
  job_role?: string
  photo_url?: string
}

export interface CreateNotificationRuleInput {
  event_type: string
  severity: 'critical' | 'warning' | 'info'
  channel: 'telegram' | 'email' | 'sms' | 'dashboard'
  target: string
}

export interface DetectionsFilter {
  camera_id?: string
  event_type?: string
  severity?: string
  limit?: number
}

// Type-safe API helpers
export const apiClient = {
  // Auth
  login: (email: string, password: string): Promise<LoginResponse> =>
    api.post('/auth/login', { email, password }).then(r => r.data),

  getMe: (): Promise<{ user: User }> =>
    api.get('/auth/me').then(r => r.data),

  // Cameras
  listCameras: (): Promise<{ cameras: Camera[] }> =>
    api.get('/vision/cameras').then(r => r.data),

  getCamera: (id: string): Promise<Camera> =>
    api.get(`/vision/cameras/${id}`).then(r => r.data),

  createCamera: (input: CreateCameraInput): Promise<Camera> =>
    api.post('/vision/cameras', input).then(r => r.data),

  // Detections
  listDetections: (filter: DetectionsFilter = {}): Promise<{ detections: Detection[] }> =>
    api.get('/vision/detections', { params: filter }).then(r => r.data),

  // Identity
  listPersons: (kind?: string): Promise<{ persons: Person[] }> =>
    api.get('/identity/persons', { params: { kind } }).then(r => r.data),

  enrollPerson: (input: EnrollPersonInput): Promise<Person> =>
    api.post('/identity/persons', input).then(r => r.data),

  matchFace: (embedding: number[], threshold: number): Promise<MatchResult> =>
    api.post('/identity/match', { embedding, threshold }).then(r => r.data),

  // Notifications
  listNotificationRules: (): Promise<{ rules: NotificationRule[] }> =>
    api.get('/notifications/rules').then(r => r.data),

  createNotificationRule: (input: CreateNotificationRuleInput): Promise<NotificationRule> =>
    api.post('/notifications/rules', input).then(r => r.data),

  // Staff
  listStaff: (): Promise<{ staff: StaffProfile[] }> =>
    api.get('/staff/').then(r => r.data),

  getStaff: (id: string): Promise<StaffProfile> =>
    api.get(`/staff/${id}`).then(r => r.data),

  getStaffReport: (id: string): Promise<{ report: string }> =>
    api.get(`/staff/${id}/report`).then(r => r.data),
}

// Import the existing axios instance
import api from '../services/api'