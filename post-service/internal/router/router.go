package route

import (
	"log/slog"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	httpDelivery "post-service/internal/delivery/http"
	"post-service/internal/usecase"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// New инициализирует chi.Router
func New(
	log *slog.Logger,
	postUC *usecase.PostUsecase,
	// commentUC *usecase.CommentUsecase,
) *chi.Mux {
	r := chi.NewRouter()

	// Middlewares
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.URLFormat)

	// Handlers
	r.Handle("/metrics", promhttp.Handler())
	postHandler := httpDelivery.NewPostHandler(log, postUC)
	// commentHandler := httpDelivery.NewCommentHandler(log, commentUC)

	// Routes
	r.Route("/posts", func(r chi.Router) {
		r.Post("/", postHandler.CreatePost)
	})

	// r.Route("/comments", func(r chi.Router) {
	// 	r.Post("/", commentHandler.CreateComment)
	// })

	return r
}
