package platform

import (
	"ctf-recruit/backend/internal/config"
	"ctf-recruit/backend/internal/middleware"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type AppContext struct {
	App *fiber.App
	DB  *gorm.DB
	Cfg config.Config
}

func NewApp(cfg config.Config) (*AppContext, error) {
	db, err := gorm.Open(postgres.Open(cfg.DatabaseURL), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	app := fiber.New(fiber.Config{ErrorHandler: middleware.ErrorHandler})
	app.Use(middleware.RequestID())
	app.Use(cors.New(cors.Config{
		AllowOrigins: "http://127.0.0.1:3001,http://localhost:3001",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
		AllowMethods: "GET,POST,PUT,DELETE,OPTIONS",
	}))

	return &AppContext{App: app, DB: db, Cfg: cfg}, nil
}
