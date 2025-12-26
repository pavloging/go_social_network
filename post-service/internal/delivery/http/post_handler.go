package http

import (
	"net/http"

	"log/slog"
	"post-service/internal/usecase"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type PostHandler struct {
	log *slog.Logger
	uc  *usecase.PostUsecase
}

func NewPostHandler(log *slog.Logger, uc *usecase.PostUsecase) *PostHandler {
	return &PostHandler{log: log, uc: uc}
}

// GET /posts/{id}
func (h *PostHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	const fn = "PostHandler.GetByID"

	log := h.log.With(
		slog.String("fn", fn),
		slog.String("request_id", middleware.GetReqID(r.Context())),
	)

	idParam := chi.URLParam(r, "id")
	if idParam == "" {
		log.Warn("empty id parameter")
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{
			"error":   "Bad request",
			"message": "ID is required",
		})
		return
	}

	log.Info("getting post", slog.String("id", idParam))

	post, err := h.uc.GetByID(r.Context(), idParam)
	if err != nil {
		log.Error("failed to get post", slog.Any("error", err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{
			"error":   "Internal server error",
			"message": "Failed to retrieve post",
		})
		return
	}

	if post == nil {
		log.Warn("post not found", slog.String("id", idParam))
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, map[string]string{
			"error":   "Not found",
			"message": "Post not found",
		})
		return
	}

	log.Info("post retrieved", slog.String("id", idParam))
	render.JSON(w, r, post)
}

// GET /posts
func (h *PostHandler) List(w http.ResponseWriter, r *http.Request) {
	const fn = "PostHandler.List"

	log := h.log.With(
		slog.String("fn", fn),
		slog.String("request_id", middleware.GetReqID(r.Context())),
	)

	log.Info("listing posts")

	posts, err := h.uc.List(r.Context())
	if err != nil {
		log.Error("failed to get posts", slog.Any("error", err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{
			"error":   "Internal server error",
			"message": "Failed to retrieve posts",
		})
		return
	}

	log.Info("posts retrieved", slog.Int("count", len(posts)))
	render.JSON(w, r, posts)
}

// POST /posts
func (h *PostHandler) CreatePost(w http.ResponseWriter, r *http.Request) {
	const fn = "PostHandler.CreatePost"

	log := h.log.With(
		slog.String("fn", fn),
		slog.String("request_id", middleware.GetReqID(r.Context())),
	)

	var req struct {
		Title   string   `json:"title"`
		Author  string   `json:"author"`
		Content string   `json:"content"`
		Tags    []string `json:"tags"`
	}

	if err := render.DecodeJSON(r.Body, &req); err != nil {
		log.Error("failed to decode request", slog.Any("error", err))
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{
			"error":   "Bad request",
			"message": "Invalid JSON payload",
		})
		return
	}

	// Валидация
	if req.Title == "" {
		log.Warn("empty title in request")
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{
			"error":   "Validation error",
			"message": "Title is required",
		})
		return
	}

	if req.Author == "" {
		log.Warn("empty author in request")
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{
			"error":   "Validation error",
			"message": "Author is required",
		})
		return
	}

	if req.Content == "" {
		log.Warn("empty content in request")
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{
			"error":   "Validation error",
			"message": "Content is required",
		})
		return
	}

	log.Info("creating post",
		slog.String("title", req.Title),
		slog.String("author", req.Author),
	)

	post, err := h.uc.CreatePost(r.Context(), req.Title, req.Author, req.Content, req.Tags)
	if err != nil {
		log.Error("failed to create post", slog.Any("error", err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{
			"error":   "Internal server error",
			"message": "Failed to create post",
		})
		return
	}

	log.Info("post created",
		slog.String("id", post.ID),
		slog.String("title", post.Title),
	)

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, post)
}
