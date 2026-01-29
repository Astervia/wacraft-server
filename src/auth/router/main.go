package auth_router

import (
	auth_handler "github.com/Astervia/wacraft-server/src/auth/handler"
	auth_middleware "github.com/Astervia/wacraft-server/src/auth/middleware"
	workspace_handler "github.com/Astervia/wacraft-server/src/workspace/handler"
	"github.com/gofiber/fiber/v2"
)

func Route(app *fiber.App) {
	group := app.Group("/auth")

	registrationRoutes(group)
	verificationRoutes(group)
	passwordResetRoutes(group)
	invitationRoutes(group)
}

func registrationRoutes(group fiber.Router) {
	group.Post("/register",
		auth_middleware.RegistrationRateLimiter,
		auth_handler.Register,
	)
}

func verificationRoutes(group fiber.Router) {
	group.Get("/verify-email", auth_handler.VerifyEmail)

	group.Post("/resend-verification",
		auth_middleware.EmailVerificationRateLimiter,
		auth_handler.ResendVerification,
	)
}

func passwordResetRoutes(group fiber.Router) {
	group.Post("/forgot-password",
		auth_middleware.PasswordResetRateLimiter,
		auth_handler.ForgotPassword,
	)

	group.Post("/reset-password", auth_handler.ResetPassword)
}

func invitationRoutes(group fiber.Router) {
	group.Post("/accept-invitation", workspace_handler.AcceptInvitation)
}
