package http

import (
	"encoding/json"
	"net/http"

	"log/slog"
	"post-service/internal/usecase"
)

type PostHandler struct {
	log *slog.Logger
	uc  *usecase.PostUsecase
}

func NewPostHandler(log *slog.Logger, uc *usecase.PostUsecase) *PostHandler {
	return &PostHandler{log: log, uc: uc}
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

	post, err := h.uc.CreatePost(req.Title, req.Author, req.Content, req.Tags)
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
