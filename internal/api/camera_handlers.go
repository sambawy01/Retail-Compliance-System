// Package api — camera_handlers.go contains camera, zone, detection, and clip HTTP handlers.
package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/sambawy01/Retail-Compliance-System/internal/vision"
)

// --- Camera handlers ---

func (s *Server) listCameras(w http.ResponseWriter, r *http.Request) {
	locationID := r.URL.Query().Get("location_id")
	cams, err := s.vision.ListCameras(r.Context(), locationID)
	if err != nil {
		slog.Error("list_cameras_failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to list cameras")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"cameras": cams})
}

func (s *Server) createCamera(w http.ResponseWriter, r *http.Request) {
	var in vision.CreateCameraInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	cam, err := s.vision.CreateCamera(r.Context(), in)
	if err != nil {
		slog.Error("create_camera_failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to create camera")
		return
	}
	writeJSON(w, http.StatusCreated, cam)
}

func (s *Server) getCamera(w http.ResponseWriter, r *http.Request) {
	cameraID := chi.URLParam(r, "cameraID")
	cam, err := s.vision.GetCamera(r.Context(), cameraID)
	if err != nil {
		writeError(w, http.StatusNotFound, "camera not found")
		return
	}
	writeJSON(w, http.StatusOK, cam)
}

func (s *Server) updateCamera(w http.ResponseWriter, r *http.Request) {
	cameraID := chi.URLParam(r, "cameraID")
	var in vision.UpdateCameraInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	cam, err := s.vision.UpdateCamera(r.Context(), cameraID, in)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update camera")
		return
	}
	writeJSON(w, http.StatusOK, cam)
}

func (s *Server) deleteCamera(w http.ResponseWriter, r *http.Request) {
	cameraID := chi.URLParam(r, "cameraID")
	if err := s.vision.DeleteCamera(r.Context(), cameraID); err != nil {
		slog.Error("delete_camera_failed", "error", err, "camera_id", cameraID)
		writeError(w, http.StatusInternalServerError, "failed to delete camera")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) cameraHeartbeat(w http.ResponseWriter, r *http.Request) {
	cameraID := chi.URLParam(r, "cameraID")
	var body struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if err := s.vision.UpdateCameraStatus(r.Context(), cameraID, vision.CameraStatus(body.Status)); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update heartbeat")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- Zone handlers ---

func (s *Server) listZones(w http.ResponseWriter, r *http.Request) {
	cameraID := r.URL.Query().Get("camera_id")
	zones, err := s.vision.ListZones(r.Context(), cameraID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list zones")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"zones": zones})
}

func (s *Server) createZone(w http.ResponseWriter, r *http.Request) {
	var in vision.CreateZoneInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	z, err := s.vision.CreateZone(r.Context(), in)
	if err != nil {
		slog.Error("create_zone_failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to create zone")
		return
	}
	writeJSON(w, http.StatusCreated, z)
}

func (s *Server) deleteZone(w http.ResponseWriter, r *http.Request) {
	zoneID := chi.URLParam(r, "zoneID")
	if err := s.vision.DeleteZone(r.Context(), zoneID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete zone")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- Detection handlers ---

func (s *Server) listDetections(w http.ResponseWriter, r *http.Request) {
	dets, err := s.vision.ListDetections(r.Context(), vision.ListDetectionsFilter{
		CameraID:  r.URL.Query().Get("camera_id"),
		EventType: r.URL.Query().Get("event_type"),
		Severity:  r.URL.Query().Get("severity"),
		Limit:     100,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list detections")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"detections": dets})
}

func (s *Server) insertDetection(w http.ResponseWriter, r *http.Request) {
	var in vision.InsertDetectionInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	det, err := s.vision.InsertDetection(r.Context(), in)
	if err != nil {
		slog.Error("insert_detection_failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to insert detection")
		return
	}
	writeJSON(w, http.StatusCreated, det)
}

// --- Clip handlers ---

func (s *Server) listClips(w http.ResponseWriter, r *http.Request) {
	cameraID := r.URL.Query().Get("camera_id")
	if cameraID == "" {
		writeJSON(w, http.StatusOK, map[string]any{"clips": []any{}})
		return
	}
	clips, err := s.vision.ListClips(r.Context(), cameraID, 100)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list clips")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"clips": clips})
}

func (s *Server) insertClip(w http.ResponseWriter, r *http.Request) {
	var in vision.InsertClipInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	clip, err := s.vision.InsertClip(r.Context(), in)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to insert clip")
		return
	}
	writeJSON(w, http.StatusCreated, clip)
}

func (s *Server) getClip(w http.ResponseWriter, r *http.Request) {
	clipID := chi.URLParam(r, "clipID")
	clip, err := s.vision.GetClip(r.Context(), clipID)
	if errors.Is(err, vision.ErrClipNotFound) {
		writeError(w, http.StatusNotFound, "clip not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get clip")
		return
	}
	writeJSON(w, http.StatusOK, clip)
}

func (s *Server) getClipURL(w http.ResponseWriter, r *http.Request) {
	clipID := chi.URLParam(r, "clipID")
	clip, err := s.vision.GetClip(r.Context(), clipID)
	if err != nil {
		writeError(w, http.StatusNotFound, "clip not found")
		return
	}
	url, err := s.vision.GeneratePresignURL(r.Context(), clip, 15)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate URL")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"url": url})
}