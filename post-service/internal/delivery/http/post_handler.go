package http

import (
	"context"
	"encoding/json"
	"net/http"

	"log/slog"
	"post-service/internal/usecase"

	"github.com/go-chi/chi/v5"
)

type PostHandler struct {
	log *slog.Logger
	uc  *usecase.PostUsecase
	ctx context.Context
}

func NewPostHandler(ctx context.Context, log *slog.Logger, uc *usecase.PostUsecase) *PostHandler {
	return &PostHandler{ctx: ctx, log: log, uc: uc}
}

// GET /posts
func (h *PostHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id") // строковый id из URL
	if idParam == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}

	h.log.Info("request is valid")

	post, err := h.uc.GetByID(h.log, h.ctx, idParam)
	if err != nil {
		h.log.Error("failed to get post", "err", err)
		http.Error(w, "failed to get post", http.StatusInternalServerError)
		return
	}

	h.log.Info("post is was get to db")

	// отдаём post обратно клиенту
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(post)
}

// GET /posts
func (h *PostHandler) List(w http.ResponseWriter, r *http.Request) {
	h.log.Info("request is valid")

	post, err := h.uc.List(h.ctx)
	if err != nil {
		h.log.Error("failed to get posts", "err", err)
		http.Error(w, "failed to get posts", http.StatusInternalServerError)
		return
	}

	h.log.Info("posts is was get to db")

	// отдаём posts обратно клиенту
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(post)
}

// POST /posts
func (h *PostHandler) CreatePost(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Title   string   `json:"title"`
		Author  string   `json:"author"`
		Content string   `json:"content"`
		Tags    []string `json:"tags"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	h.log.Info("request is valid")

	post, err := h.uc.CreatePost(h.ctx, req.Title, req.Author, req.Content, req.Tags)
	if err != nil {
		h.log.Error("failed to create post", "err", err)
		http.Error(w, "failed to create post", http.StatusInternalServerError)
		return
	}

	h.log.Info("message is was send to notification-service")

	// отдаём post обратно клиенту
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(post)
}
