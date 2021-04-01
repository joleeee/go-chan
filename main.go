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

	//if err := db.Update(
		//func(tx *nutsdb.Tx) error {
			//key := []byte("id")
			//val := []byte("0")
			//bucket := "internal"
			//if err := tx.Put(bucket, key, val, 0); err != nil {
				//return err
			//}
			//return nil
	//}); err != nil {
		//panic(1)
	//}

	handlers.Init(db)

	e := echo.New()

	//e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	e.Static("/", "sauce")
	e.GET("/", handlers.Root)
	e.GET("/posts", handlers.Posts)
	e.GET("/posts/:data", handlers.Post)
	e.POST("/posts/newpost", handlers.NewPost)
	e.POST("/posts/newthread", handlers.NewThread)

	e.Logger.Fatal(e.Start("localhost:4242"))
}

