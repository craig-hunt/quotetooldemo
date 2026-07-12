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
	store Store
	log   *slog.Logger
}

func NewHandlers(store Store, log *slog.Logger) *Handlers {
	return &Handlers{store: store, log: log}
}

func (h *Handlers) Routes() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", h.health)
	mux.HandleFunc("POST /quotes", h.create)
	mux.HandleFunc("GET /quotes", h.list)
	mux.HandleFunc("GET /quotes/{id}", h.get)
	mux.HandleFunc("PUT /quotes/{id}", h.update)
	mux.HandleFunc("POST /quotes/{id}/send", h.transitionTo(StatusSent))
	mux.HandleFunc("POST /quotes/{id}/accept", h.transitionTo(StatusAccepted))
	mux.HandleFunc("POST /quotes/{id}/reject", h.transitionTo(StatusRejected))

	return mux
}

func (h *Handlers) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{StatusKey: StatusOK})
}

func (h *Handlers) create(w http.ResponseWriter, r *http.Request) {
	var in QuoteInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, ErrMsgInvalidJSON)
		return
	}
	if err := validateQuoteInput(in); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	q, err := h.store.Create(r.Context(), in)
	if err != nil {
		h.log.Error("create quote", "err", err)
		writeError(w, http.StatusInternalServerError, ErrMsgCreateFailed)
		return
	}
	writeJSON(w, http.StatusCreated, q)
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
	if v := r.URL.Query().Get(QueryParamCustomerID); v != "" {
		if id, err := uuid.Parse(v); err == nil {
			p.CustomerID = &id
		}
	}
	if v := r.URL.Query().Get(QueryParamStatus); v != "" {
		s := QuoteStatus(v)
		p.Status = &s
	}

	qs, err := h.store.List(r.Context(), p)
	if err != nil {
		h.log.Error("list quotes", "err", err)
		writeError(w, http.StatusInternalServerError, ErrMsgListFailed)
		return
	}
	writeJSON(w, http.StatusOK, qs)
}

func (h *Handlers) get(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	q, err := h.store.Get(r.Context(), id)
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, ErrMsgNotFound)
		return
	}
	if err != nil {
		h.log.Error("get quote", "id", id, "err", err)
		writeError(w, http.StatusInternalServerError, ErrMsgGetFailed)
		return
	}
	writeJSON(w, http.StatusOK, q)
}

func (h *Handlers) update(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	var in QuoteInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, ErrMsgInvalidJSON)
		return
	}
	if err := validateQuoteInput(in); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	q, err := h.store.Update(r.Context(), id, in)
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, ErrMsgNotFound)
		return
	}
	if err != nil {
		h.log.Error("update quote", "id", id, "err", err)
		writeError(w, http.StatusInternalServerError, ErrMsgUpdateFailed)
		return
	}
	writeJSON(w, http.StatusOK, q)
}

// transitionTo returns a handler function bound to a specific target status.
// Closure captures `to`; each of send/accept/reject registers its own handler.
func (h *Handlers) transitionTo(to QuoteStatus) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := parseID(w, r)
		if !ok {
			return
		}

		q, err := h.store.Transition(r.Context(), id, to)
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
		writeJSON(w, http.StatusOK, q)
	}
}

func validateQuoteInput(in QuoteInput) error {
	if in.CustomerID == uuid.Nil {
		return errors.New(ErrMsgCustomerIDRequired)
	}
	if in.ProjectName == "" {
		return errors.New(ErrMsgProjectNameRequired)
	}
	if len(in.LineItems) == 0 {
		return errors.New(ErrMsgLineItemsRequired)
	}
	for i, li := range in.LineItems {
		if li.AreaSqft <= 0 {
			return errors.New(ErrMsgLinePrefix + strconv.Itoa(i) + ": " + ErrMsgInvalidArea)
		}
		if li.DepthInches <= 0 {
			return errors.New(ErrMsgLinePrefix + strconv.Itoa(i) + ": " + ErrMsgInvalidDepth)
		}
		if li.UnitPricePerTon <= 0 {
			return errors.New(ErrMsgLinePrefix + strconv.Itoa(i) + ": " + ErrMsgInvalidUnitPrice)
		}
		switch li.MixType {
		case MixHMABase, MixHMASurface, MixSuperpave, MixWarmMix:
		default:
			return errors.New(ErrMsgLinePrefix + strconv.Itoa(i) + ": " + ErrMsgInvalidMixType)
		}
	}
	return nil
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
