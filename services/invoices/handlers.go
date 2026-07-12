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
	ordersClient *OrdersClient
	log          *slog.Logger
}

func NewHandlers(store Store, oc *OrdersClient, log *slog.Logger) *Handlers {
	return &Handlers{store: store, ordersClient: oc, log: log}
}

func (h *Handlers) Routes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", h.health)
	mux.HandleFunc("POST /invoices", h.create)
	mux.HandleFunc("GET /invoices", h.list)
	mux.HandleFunc("GET /invoices/{id}", h.get)
	mux.HandleFunc("POST /invoices/{id}/send", h.transitionTo(StatusSent))
	mux.HandleFunc("POST /invoices/{id}/mark_paid", h.transitionTo(StatusPaid))
	mux.HandleFunc("POST /invoices/{id}/cancel", h.transitionTo(StatusCancelled))
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
	if in.OrderID == uuid.Nil {
		writeError(w, http.StatusBadRequest, ErrMsgOrderIDRequired)
		return
	}
	if in.DueDays <= 0 {
		in.DueDays = DefaultDueDays
	}

	o, err := h.ordersClient.Get(r.Context(), in.OrderID)
	if errors.Is(err, ErrOrderNotFound) {
		writeError(w, http.StatusBadRequest, ErrMsgOrderNotFound)
		return
	}
	if err != nil {
		h.log.Error("fetch order", "err", err)
		writeError(w, http.StatusBadGateway, ErrMsgOrdersUnavailable)
		return
	}
	if o.Status != ExternalOrderStatusFulfilled {
		writeError(w, http.StatusConflict, ErrMsgOrderNotFulfilled)
		return
	}

	inv, err := h.store.Create(r.Context(), in, o)
	if err != nil {
		h.log.Error("create invoice", "err", err)
		writeError(w, http.StatusInternalServerError, ErrMsgCreateFailed)
		return
	}
	writeJSON(w, http.StatusCreated, inv)
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
		s := InvoiceStatus(v)
		p.Status = &s
	}

	invs, err := h.store.List(r.Context(), p)
	if err != nil {
		h.log.Error("list invoices", "err", err)
		writeError(w, http.StatusInternalServerError, ErrMsgListFailed)
		return
	}
	writeJSON(w, http.StatusOK, invs)
}

func (h *Handlers) get(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	inv, err := h.store.Get(r.Context(), id)
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, ErrMsgNotFound)
		return
	}
	if err != nil {
		h.log.Error("get invoice", "id", id, "err", err)
		writeError(w, http.StatusInternalServerError, ErrMsgGetFailed)
		return
	}
	writeJSON(w, http.StatusOK, inv)
}

func (h *Handlers) transitionTo(to InvoiceStatus) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := parseID(w, r)
		if !ok {
			return
		}
		inv, err := h.store.Transition(r.Context(), id, to)
		if errors.Is(err, ErrNotFound) {
			writeError(w, http.StatusNotFound, ErrMsgNotFound)
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
		writeJSON(w, http.StatusOK, inv)
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
