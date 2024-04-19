package vscode

import (
	"github.com/labstack/echo/v4"
)

func InitRouter(router *echo.Group) {
	handler := NewVscodeHandler()
	vscode := router.Group("/vscode")
	{
		vscode.POST("/watch", handler.watch)
		vscode.GET("/state", handler.state)
		vscode.GET("/readDirectory", handler.readDirectory)
		vscode.POST("/createDirectory", handler.createDirectory)
		vscode.GET("/readFile", handler.readFile)
		vscode.POST("/writeFile", handler.writeFile)
		vscode.DELETE("/delete", handler.delete)
		vscode.PUT("/rename", handler.rename)
		vscode.POST("/copy", handler.copy)
	}
}
