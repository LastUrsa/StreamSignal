package sip

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

func (s *Service) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/app", s.handleApp)
	mux.HandleFunc("/api/v1/health", s.handleHealth)
	mux.HandleFunc("/api/v1/capabilities", s.handleCapabilities)
	mux.HandleFunc("/api/v1/status", s.handleStatus)
	mux.HandleFunc("/api/v1/profiles", s.handleProfiles)
	mux.HandleFunc("/api/v1/profile", s.handleProfile)
	return securityHeaders(requireLocalHost(mux))
}

func (s *Service) handleApp(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	response, err := s.App(r.Context())
	writeResult(w, response, err)
}

func (s *Service) handleHealth(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	response, err := s.Health(r.Context())
	writeResult(w, response, err)
}

func (s *Service) handleCapabilities(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	response, err := s.Capabilities(r.Context())
	writeResult(w, response, err)
}

func (s *Service) handleStatus(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	response, err := s.Status(r.Context())
	writeResult(w, response, err)
}

func (s *Service) handleProfiles(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	response, err := s.Profiles(r.Context())
	writeResult(w, response, err)
}

func (s *Service) handleProfile(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	if !requireJSON(w, r) {
		return
	}
	defer r.Body.Close()
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	var request ActivateProfileRequest
	if err := decoder.Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, ErrInvalidRequest)
		return
	}
	response, err := s.ActivateProfile(r.Context(), request.Profile)
	writeResult(w, response, err)
}

func requireLocalHost(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := hostWithoutPort(r.Host)
		if host != "127.0.0.1" && host != "localhost" {
			writeError(w, http.StatusForbidden, ErrForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}

func requireMethod(w http.ResponseWriter, r *http.Request, method string) bool {
	if r.Method == method {
		return true
	}
	w.Header().Set("Allow", method)
	writeError(w, http.StatusMethodNotAllowed, errors.New("InvalidRequest"))
	return false
}

func requireJSON(w http.ResponseWriter, r *http.Request) bool {
	contentType := r.Header.Get("Content-Type")
	if contentType == "" {
		writeError(w, http.StatusUnsupportedMediaType, ErrInvalidRequest)
		return false
	}
	mediaType := strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
	if mediaType != "application/json" {
		writeError(w, http.StatusUnsupportedMediaType, ErrInvalidRequest)
		return false
	}
	return true
}

func hostWithoutPort(host string) string {
	host = strings.TrimSpace(strings.ToLower(host))
	if strings.HasPrefix(host, "[") {
		end := strings.Index(host, "]")
		if end >= 0 {
			return strings.Trim(host[:end+1], "[]")
		}
		return host
	}
	if index := strings.LastIndex(host, ":"); index > -1 {
		return host[:index]
	}
	return host
}

func writeResult(w http.ResponseWriter, response any, err error) {
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidRequest):
			writeError(w, http.StatusBadRequest, err)
		case errors.Is(err, ErrProfileNotFound):
			writeError(w, http.StatusNotFound, err)
		case errors.Is(err, ErrForbidden):
			writeError(w, http.StatusForbidden, err)
		default:
			writeError(w, http.StatusInternalServerError, err)
		}
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func writeError(w http.ResponseWriter, status int, err error) {
	message := "UnexpectedFailure"
	if err != nil && err.Error() != "" {
		message = err.Error()
	}
	writeJSON(w, status, ErrorResponse{Success: false, Error: message})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
