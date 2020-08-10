package main

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	InitConfig()
	InitDB()

	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// Debug route
	e.GET("/ping", PingHandler)

	// Real route
	e.POST("/api/v1/decrypt", DecryptDataHandler)

	e.Logger.Fatal(e.Start(CFG.SeverHost + ":" + CFG.ServerPort))
}