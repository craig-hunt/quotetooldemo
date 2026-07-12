package main

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/google/uuid"
)

// Handlers wires HTTP endpoints to the Store.
type Handlers struct {
	store Store
	log   *slog.Logger
}

func NewHandlers(store Store, log *slog.Logger) *Handlers {
	return &Handlers{store: store, log: log}
}

// Routes returns a mux with the customer endpoints wired.
// Go 1.22 introduced pattern-based routing on net/http, which removes
// the need for chi or gorilla/mux for simple CRUD services.
func (h *Handlers) Routes() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", h.health)
	mux.HandleFunc("POST /customers", h.create)
	mux.HandleFunc("GET /customers", h.list)
	mux.HandleFunc("GET /customers/{id}", h.get)
	mux.HandleFunc("PUT /customers/{id}", h.update)
	mux.HandleFunc("DELETE /customers/{id}", h.delete)

	return mux
}

func (h *Handlers) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{StatusKey: StatusOK})
}

func (h *Handlers) create(w http.ResponseWriter, r *http.Request) {
	var in CustomerInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, ErrMsgInvalidJSON)
		return
	}
	if in.Name == "" {
		writeError(w, http.StatusBadRequest, ErrMsgNameRequired)
		return
	}

	c, err := h.store.Create(r.Context(), in)
	if err != nil {
		h.log.Error("create customer", "err", err)
		writeError(w, http.StatusInternalServerError, ErrMsgCreateFailed)
		return
	}
	writeJSON(w, http.StatusCreated, c)
}

func (h *Handlers) list(w http.ResponseWriter, r *http.Request) {
	p := ListParams{Limit: DefaultPageLimit, Offset: 0}
	if v := r.URL.Query().Get(QueryParamLimit); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= MaxPageLimit {
			p.Limit = n
		}
	}
	if v := r.URL.Query().Get(QueryParamOffset); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			p.Offset = n
		}
	}

	cs, err := h.store.List(r.Context(), p)
	if err != nil {
		h.log.Error("list customers", "err", err)
		writeError(w, http.StatusInternalServerError, ErrMsgListFailed)
		return
	}
	writeJSON(w, http.StatusOK, cs)
}

func (h *Handlers) get(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	c, err := h.store.Get(r.Context(), id)
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, ErrMsgNotFound)
		return
	}
	if err != nil {
		h.log.Error("get customer", "id", id, "err", err)
		writeError(w, http.StatusInternalServerError, ErrMsgGetFailed)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (h *Handlers) update(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	var in CustomerInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, ErrMsgInvalidJSON)
		return
	}
	if in.Name == "" {
		writeError(w, http.StatusBadRequest, ErrMsgNameRequired)
		return
	}

	c, err := h.store.Update(r.Context(), id, in)
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, ErrMsgNotFound)
		return
	}
	if err != nil {
		h.log.Error("update customer", "id", id, "err", err)
		writeError(w, http.StatusInternalServerError, ErrMsgUpdateFailed)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (h *Handlers) delete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	if err := h.store.Delete(r.Context(), id); errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, ErrMsgNotFound)
		return
	} else if err != nil {
		h.log.Error("delete customer", "id", id, "err", err)
		writeError(w, http.StatusInternalServerError, ErrMsgDeleteFailed)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// parseID extracts a UUID from the {id} path parameter.
// Returns ok=false and writes a 400 if the parameter fails to parse.
func parseID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	raw := r.PathValue("id")
	id, err := uuid.Parse(raw)
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrMsgInvalidID)
		return uuid.Nil, false
	}
	return id, true
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set(HeaderContentType, ContentTypeJSON)
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{ErrorKey: msg})
}
