package main

import (
	"os"

	"github.com/gonflix/employee-manage/internal/db"
	"github.com/gonflix/employee-manage/internal/handler"
	"github.com/gonflix/employee-manage/internal/repository"
	"github.com/gonflix/employee-manage/internal/service"
	"github.com/labstack/echo/v4"
)

func main() {
	pdb := db.ConnectPDB()

	repository := repository.NewEmployeeRepository(pdb)
	service := service.NewEmployeeService(repository)
	handler := handler.NewEmployeeHandler(service)

	e := echo.New()
	handler.RegisterRoutes(e)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	e.Logger.Fatal(e.Start(":" + port))
}
