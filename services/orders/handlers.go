package main

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/google/uuid"
)

type Handlers struct {
	store        Store
	quotesClient *QuotesClient
	log          *slog.Logger
}

func NewHandlers(store Store, qc *QuotesClient, log *slog.Logger) *Handlers {
	return &Handlers{store: store, quotesClient: qc, log: log}
}

func (h *Handlers) Routes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", h.health)
	mux.HandleFunc("POST /orders", h.create)
	mux.HandleFunc("GET /orders", h.list)
	mux.HandleFunc("GET /orders/{id}", h.get)
	mux.HandleFunc("PUT /orders/{id}", h.update)
	mux.HandleFunc("POST /orders/{id}/fulfill", h.transitionTo(StatusFulfilled))
	mux.HandleFunc("POST /orders/{id}/cancel", h.transitionTo(StatusCancelled))
	return mux
}

func (h *Handlers) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{StatusKey: StatusOK})
}

func (h *Handlers) create(w http.ResponseWriter, r *http.Request) {
	var in CreateInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, ErrMsgInvalidJSON)
		return
	}
	if in.QuoteID == uuid.Nil {
		writeError(w, http.StatusBadRequest, ErrMsgQuoteIDRequired)
		return
	}

	// Cross-service call: fetch the quote to validate + copy fields.
	q, err := h.quotesClient.Get(r.Context(), in.QuoteID)
	if errors.Is(err, ErrQuoteNotFound) {
		writeError(w, http.StatusBadRequest, ErrMsgQuoteNotFound)
		return
	}
	if err != nil {
		h.log.Error("fetch quote", "err", err)
		writeError(w, http.StatusBadGateway, ErrMsgQuotesUnavailable)
		return
	}
	if q.Status != ExternalQuoteStatusAccepted {
		writeError(w, http.StatusConflict, ErrMsgQuoteNotAccepted)
		return
	}

	o, err := h.store.Create(r.Context(), in, q)
	if err != nil {
		h.log.Error("create order", "err", err)
		writeError(w, http.StatusInternalServerError, ErrMsgCreateFailed)
		return
	}
	writeJSON(w, http.StatusCreated, o)
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
	if v := r.URL.Query().Get(QueryParamStatus); v != "" {
		s := OrderStatus(v)
		p.Status = &s
	}

	os, err := h.store.List(r.Context(), p)
	if err != nil {
		h.log.Error("list orders", "err", err)
		writeError(w, http.StatusInternalServerError, ErrMsgListFailed)
		return
	}
	writeJSON(w, http.StatusOK, os)
}

func (h *Handlers) get(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	o, err := h.store.Get(r.Context(), id)
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, ErrMsgOrderNotFound)
		return
	}
	if err != nil {
		h.log.Error("get order", "id", id, "err", err)
		writeError(w, http.StatusInternalServerError, ErrMsgGetFailed)
		return
	}
	writeJSON(w, http.StatusOK, o)
}

func (h *Handlers) update(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var in UpdateInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, ErrMsgInvalidJSON)
		return
	}
	o, err := h.store.Update(r.Context(), id, in)
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, ErrMsgOrderNotFound)
		return
	}
	if err != nil {
		h.log.Error("update order", "id", id, "err", err)
		writeError(w, http.StatusInternalServerError, ErrMsgUpdateFailed)
		return
	}
	writeJSON(w, http.StatusOK, o)
}

func (h *Handlers) transitionTo(to OrderStatus) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := parseID(w, r)
		if !ok {
			return
		}
		o, err := h.store.Transition(r.Context(), id, to)
		if errors.Is(err, ErrNotFound) {
			writeError(w, http.StatusNotFound, ErrMsgOrderNotFound)
			return
		}
		if errors.Is(err, ErrInvalidStatus) {
			writeError(w, http.StatusConflict, ErrMsgInvalidTransition)
			return
		}
		if err != nil {
			h.log.Error("transition", "id", id, "to", to, "err", err)
			writeError(w, http.StatusInternalServerError, ErrMsgTransitionFailed)
			return
		}
		writeJSON(w, http.StatusOK, o)
	}
}

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
