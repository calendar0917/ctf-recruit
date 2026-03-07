package router

import (
	"ctf-recruit/backend/internal/middleware"
	"ctf-recruit/backend/internal/modules/announcement"
	"ctf-recruit/backend/internal/modules/auth"
	"ctf-recruit/backend/internal/modules/challenge"
	"ctf-recruit/backend/internal/modules/health"
	"ctf-recruit/backend/internal/modules/instance"
	"ctf-recruit/backend/internal/modules/judge"
	"ctf-recruit/backend/internal/modules/recruitment"
	"ctf-recruit/backend/internal/modules/scoreboard"
	"ctf-recruit/backend/internal/modules/submission"
	"ctf-recruit/backend/internal/platform"
	"time"
)

func Register(appCtx *platform.AppContext) {
	h := health.NewHandler()

	authRepo := auth.NewRepository(appCtx.DB)
	authService := auth.NewService(authRepo, appCtx.Cfg.JWTSecret, appCtx.Cfg.JWTTTL)
	authHandler := auth.NewHandler(authService)
	loginRateLimiter := middleware.NewRateLimiter(5, time.Minute)
	submissionRateLimiter := middleware.NewRateLimiter(8, time.Minute)

	challengeRepo := challenge.NewRepository(appCtx.DB)
	challengeService := challenge.NewService(challengeRepo)
	challengeHandler := challenge.NewHandler(challengeService)

	judgeRepo := judge.NewRepository(appCtx.DB)
	judgeQueue := judge.NewQueue(judgeRepo)

	submissionRepo := submission.NewRepository(appCtx.DB)
	submissionService := submission.NewService(submissionRepo, challengeService, judgeQueue)
	submissionHandler := submission.NewHandler(submissionService)

	scoreboardRepo := scoreboard.NewRepository(appCtx.DB)
	scoreboardService := scoreboard.NewService(scoreboardRepo)
	scoreboardHandler := scoreboard.NewHandler(scoreboardService)

	announcementRepo := announcement.NewRepository(appCtx.DB)
	announcementService := announcement.NewService(announcementRepo)
	announcementHandler := announcement.NewHandler(announcementService)

	recruitmentRepo := recruitment.NewRepository(appCtx.DB)
	recruitmentService := recruitment.NewService(recruitmentRepo)
	recruitmentHandler := recruitment.NewHandler(recruitmentService)

	instanceRepo := instance.NewRepository(appCtx.DB)
	runtimeController := instance.NewDockerController(appCtx.Cfg.InstanceAccessHost)
	instanceService := instance.NewServiceWithRuntime(instanceRepo, challengeService, runtimeController)
	instanceHandler := instance.NewHandler(instanceService)

	api := appCtx.App.Group("/api")
	v1 := api.Group("/v1")

	v1.Get("/health", h.GetHealth)

	authGroup := v1.Group("/auth")
	authGroup.Post("/register", authHandler.Register)
	authGroup.Post("/login", loginRateLimiter.Middleware(), authHandler.Login)
	authGroup.Get("/me", middleware.Auth(authService), authHandler.Me)
	authGroup.Get("/admin-sample", middleware.Auth(authService), middleware.RequireRoles(auth.RoleAdmin), authHandler.AdminSample)

	adminUsers := v1.Group("/admin/users", middleware.Auth(authService), middleware.RequireRoles(auth.RoleAdmin))
	adminUsers.Get("", authHandler.AdminListUsers)
	adminUsers.Patch("/:id", authHandler.AdminUpdateUser)

	challengeHandler.RegisterRoutes(v1, authService)
	submissionHandler.RegisterRoutes(v1, authService, submissionRateLimiter.Middleware())
	scoreboardHandler.RegisterRoutes(v1, authService)
	announcementHandler.RegisterRoutes(v1, authService)
	recruitmentHandler.RegisterRoutes(v1, authService)
	instanceHandler.RegisterRoutes(v1, authService)
}
