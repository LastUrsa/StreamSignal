package sip

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
)

func (s *Service) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/app", s.handleApp)
	mux.HandleFunc("/api/v1/health", s.handleHealth)
	mux.HandleFunc("/api/v1/capabilities", s.handleCapabilities)
	mux.HandleFunc("/api/v1/status", s.handleStatus)
	mux.HandleFunc("/api/v1/announcement-fields", s.handleAnnouncementFields)
	mux.HandleFunc("/api/v1/profiles", s.handleProfiles)
	mux.HandleFunc("/api/v1/profile/current", s.handleCurrentProfile)
	mux.HandleFunc("/api/v1/profile", s.handleProfile)
	mux.HandleFunc("/api/v1/announce/status", s.handleAnnounceStatus)
	mux.HandleFunc("/api/v1/announce/confirm", s.handleAnnounceConfirm)
	mux.HandleFunc("/api/v1/announce", s.handleAnnounce)
	mux.HandleFunc("/api/v1/end-stream/status", s.handleEndStreamStatus)
	mux.HandleFunc("/api/v1/end-stream", s.handleEndStream)
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

func (s *Service) handleAnnouncementFields(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		response, err := s.AnnouncementFields(r.Context())
		writeResult(w, response, err)
	case http.MethodPost:
		if !requireJSON(w, r) {
			return
		}
		defer r.Body.Close()
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
		decoder.DisallowUnknownFields()
		var request UpdateAnnouncementFieldsRequest
		if err := decoder.Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, ErrInvalidRequest)
			return
		}
		var extra struct{}
		if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
			writeError(w, http.StatusBadRequest, ErrInvalidRequest)
			return
		}
		response, err := s.UpdateAnnouncementFields(r.Context(), request)
		writeResult(w, response, err)
	default:
		w.Header().Set("Allow", strings.Join([]string{http.MethodGet, http.MethodPost}, ", "))
		writeError(w, http.StatusMethodNotAllowed, ErrInvalidRequest)
	}
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

func (s *Service) handleCurrentProfile(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	response, err := s.CurrentProfile(r.Context())
	writeResult(w, response, err)
}

func (s *Service) handleAnnounce(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	request, ok := readAnnounceRequest(w, r)
	if !ok {
		return
	}
	response, err := s.Announce(r.Context(), request)
	writeResult(w, response, err)
}

func (s *Service) handleAnnounceStatus(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	response, err := s.AnnounceStatus(r.Context())
	writeResult(w, response, err)
}

func (s *Service) handleAnnounceConfirm(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	if !requireJSON(w, r) {
		return
	}
	defer r.Body.Close()
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	var request AnnounceConfirmRequest
	if err := decoder.Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, ErrInvalidRequest)
		return
	}
	var extra struct{}
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, ErrInvalidRequest)
		return
	}
	response, err := s.ConfirmAnnouncement(r.Context(), request.ConfirmationID)
	writeResult(w, response, err)
}

func (s *Service) handleEndStream(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	if !readEmptyJSONBody(w, r) {
		return
	}
	response, err := s.EndStream(r.Context())
	writeResult(w, response, err)
}

func (s *Service) handleEndStreamStatus(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	response, err := s.EndStreamStatus(r.Context())
	writeResult(w, response, err)
}

func readEmptyJSONBody(w http.ResponseWriter, r *http.Request) bool {
	if (r.Header.Get("Content-Type") != "" || r.ContentLength != 0) && !requireJSON(w, r) {
		return false
	}
	defer r.Body.Close()
	if r.Body == nil {
		return true
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	var request struct{}
	if err := decoder.Decode(&request); err != nil && !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, ErrInvalidRequest)
		return false
	}
	var extra struct{}
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, ErrInvalidRequest)
		return false
	}
	return true
}

func readAnnounceRequest(w http.ResponseWriter, r *http.Request) (AnnounceRequest, bool) {
	if (r.Header.Get("Content-Type") != "" || r.ContentLength != 0) && !requireJSON(w, r) {
		return AnnounceRequest{}, false
	}
	defer r.Body.Close()
	if r.Body == nil {
		return AnnounceRequest{}, true
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	var request AnnounceRequest
	if err := decoder.Decode(&request); err != nil && !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, ErrInvalidRequest)
		return AnnounceRequest{}, false
	}
	var extra struct{}
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, ErrInvalidRequest)
		return AnnounceRequest{}, false
	}
	return request, true
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
