package user_router

import (
	auth_middleware "github.com/Astervia/wacraft-server/src/auth/middleware"
	billing_middleware "github.com/Astervia/wacraft-server/src/billing/middleware"
	user_handler "github.com/Astervia/wacraft-server/src/user/handler"
	"github.com/gofiber/fiber/v2"
)

func Route(app *fiber.App) {
	group := app.Group("/user")

	mainRoutes(group)
	authRoutes(group)
}

func mainRoutes(group fiber.Router) {
	group.Get("/me", auth_middleware.UserMiddleware, billing_middleware.ThroughputMiddleware, user_handler.GetCurrentUser)
	group.Delete("/me", auth_middleware.UserMiddleware, billing_middleware.ThroughputMiddleware, user_handler.DeleteCurrentUser)
	group.Put("/me", auth_middleware.UserMiddleware, billing_middleware.ThroughputMiddleware, user_handler.UpdateCurrentUser)
	group.Get("/", auth_middleware.UserMiddleware, auth_middleware.SuMiddleware, billing_middleware.ThroughputMiddleware, user_handler.Get)
	group.Post("/", auth_middleware.UserMiddleware, auth_middleware.SuMiddleware, billing_middleware.ThroughputMiddleware, user_handler.CreateUser)
	group.Delete("/", auth_middleware.UserMiddleware, auth_middleware.SuMiddleware, billing_middleware.ThroughputMiddleware, user_handler.DeleteUserByID)
	group.Put("/", auth_middleware.UserMiddleware, auth_middleware.SuMiddleware, billing_middleware.ThroughputMiddleware, user_handler.UpdateUserByID)
	group.Get("/content/:keyName/like/:likeText", auth_middleware.UserMiddleware, auth_middleware.SuMiddleware, billing_middleware.ThroughputMiddleware, user_handler.ContentKeyLike)
}
