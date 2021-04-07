package main

import (
	"log"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"

	"github.com/xujiajun/nutsdb"

	"github.com/joleeee/go-chan/handlers"
)

func main(){
	opt := nutsdb.DefaultOptions
	opt.Dir = "chandb"
	db, err := nutsdb.Open(opt)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	handlers.Init("chandb")

	e := echo.New()

	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: "method=${method}, uri=${uri}, status=${status}\n",
	}))
	e.Use(middleware.Recover())

	e.Static("/", "sauce")
	e.Static("/img", "img")
	e.GET("/", handlers.Root)
	e.GET("/threads/", handlers.ThreadList)
	e.GET("/threads/:data", handlers.Thread)
	e.POST("/threads/newthread", handlers.NewPost)

	e.Logger.Fatal(e.Start("localhost:4242"))
}

