package main

import (
	"fmt"
	"os"
	"io"
	"log"
	"net/http"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"

	"github.com/xujiajun/nutsdb"
	"strconv"

	"bytes"
	"encoding/gob"

	"path/filepath"
)

var pre string = "<!DOCTYPE html><html><body>"
var post string = "</body></html>"

var db *nutsdb.DB
func getId() int64{
	var id int64

	// get id
	if err := db.View(
	func(tx *nutsdb.Tx) error{
		key := []byte("id")
		bucket := "internal"
		if e, err := tx.Get(bucket, key); err != nil {
			return err
		} else {
			str := string(e.Value)
			fmt.Println(str)

			var ierr error
			id, ierr = strconv.ParseInt(str, 10, 64)
			if ierr != nil{
				return err
			}
		}
		return nil
	}); err != nil {
		log.Fatal(err)
	}

	//set id
	if err := db.Update(
		func(tx *nutsdb.Tx) error {
			key := []byte("id")
			str := fmt.Sprintf("%d", id+1)
			val := []byte(str)
			bucket := "internal"
			if err := tx.Put(bucket, key, val, 0); err != nil {
				return err
			}
			return nil
	}); err != nil {
		log.Fatal(err)
	}

	fmt.Println("id is", id)
	return id
}

type Message struct {
	Name string
	Subject string
	Content string
}

func message(id int64, parent int64, msg Message){
	var network bytes.Buffer
	enc := gob.NewEncoder(&network)
	err := enc.Encode(msg)
	if err != nil {
		log.Fatal(err)
	}
	// add post
	if err := db.Update(
		func(tx *nutsdb.Tx) error {
			key := []byte(fmt.Sprintf("%d", id))
			val := network.Bytes()
			bucket := "posts"
			if err := tx.Put(bucket, key, val, 0); err != nil {
				return err
			}
			return nil
	}); err != nil {
		log.Fatal(err)
	}
	// add post as child
	if err := db.Update(
		func(tx *nutsdb.Tx) error {
			bucket := "children"
			key := []byte(fmt.Sprintf("thread_%d", parent))
			val := []byte(fmt.Sprintf("%d", id))
			return tx.RPush(bucket, key, val)
	}); err != nil {
		log.Fatal(err)
	}
}

func getPost(id int64) Message {
	var msg Message
	if err := db.View(
	func(tx *nutsdb.Tx) error{
		key := []byte(fmt.Sprintf("%d", id))
		bucket := "posts"
		if e, err := tx.Get(bucket, key); err != nil {
			return err
		} else {
			network := bytes.NewBuffer(e.Value)
			dec := gob.NewDecoder(network)
			err := dec.Decode(&msg)
			if err != nil{
				log.Fatal(err)
			}
		}
		return nil
	}); err != nil {
		log.Fatal(err)
	}
	return msg
}

func thread(id int64){
	fmt.Printf("add %d to threadlist\n", id)
	if err := db.Update(
		func(tx *nutsdb.Tx) error {
			bucket := "threads"
			key := []byte("threadlist")
			val := []byte(fmt.Sprintf("%d", id))
			return tx.RPush(bucket, key, val)
	}); err != nil {
		log.Fatal(err)
	}
}

func getThreads() []string {
	var entr [][]byte
	if err := db.View(
		func(tx *nutsdb.Tx) error {
			bucket := "threads"
			key := []byte("threadlist")
			if items, err := tx.LRange(bucket, key, 0, -1); err != nil {
				return err
			} else {
				entr = items
			}
			return nil
		}); err != nil {
			log.Fatal(err)
		}
	var list []string
	for _, e := range entr {
		list = append(list, string(e))
	}
	return list
}

func getThreadPosts(thread string) []string {
	var entr [][]byte
	if err := db.View(
		func(tx *nutsdb.Tx) error {
			bucket := "children"
			key := []byte("thread_"+thread)
			if items, err := tx.LRange(bucket, key, 0, -1); err != nil {
				return err
			} else {
				entr = items
			}
			return nil
		}); err != nil {
			log.Fatal(err)
		}
	var list []string
	for _, e := range entr {
		list = append(list, string(e))
	}
	return list
}


func main(){
	opt := nutsdb.DefaultOptions
	opt.Dir = "chandb"
	var err error
	db, err = nutsdb.Open(opt)
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

	e := echo.New()

	//e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	e.Static("/", "sauce")
	e.GET("/", rootHandler)
	e.GET("/posts", postsHandler)
	e.GET("/posts/:data", postHandler)
	e.POST("/posts/newpost", newPostHandler)
	e.POST("/posts/newthread", newThreadHandler)

	e.Logger.Fatal(e.Start("localhost:4242"))
}

func rootHandler(c echo.Context) (err error){
	return c.String(http.StatusOK, "hello");
}

func postsHandler(c echo.Context) (err error){
	format := c.QueryParam("format")

	if format == "" {
		oot := getThreads()
		s := "threadlist<br>"
		for _, e := range oot {
			//network := bytes.NewBuffer(e.Value)
			//dec := gob.NewDecoder(network)
			//var t Message
			//err := dec.Decode(&t)
			//if err != nil{
				//log.Fatal(err)
			//}
			//s2 := fmt.Sprintf("<a href=\"posts/%s\">%s</a>&emsp;%s&emsp;%s<br>", string(e.Key), string(e.Key), t.Name, t.Subject)
			s2 := fmt.Sprintf("<a href=\"posts/%s\">%s</a><br>", e, e)
			s += s2
		}
		return c.HTML(http.StatusOK, s)
	} else if format == "raw" {
		return c.String(http.StatusOK, "rawr")
	} else {
		return c.String(http.StatusBadRequest, "format unrecognized\n")
	}
}

func postHandler(c echo.Context) (err error){
	nr := c.Param("data")

	a := getThreadPosts(nr)

	s := ""
	for _, e := range a{
		id, err := strconv.ParseInt(e, 10, 64)
		if err != nil {
			log.Fatal(err)
		}
		msg := getPost(id)
		post := fmt.Sprintf(`
		<hr> 
		<div class="post">
			<span class="titlebar">
				<span class="author">
					%s
				</span>
			</span>	
			<br>
			<span class="postcontent">
				%s
			</span>
		</div>`, msg.Name, msg.Content)
		s += post
	}

	return c.HTML(http.StatusOK, "rawr"+nr+"<br>"+s)
}

func newPostHandler(c echo.Context) (err error){
	thread := c.QueryParam("thread")
	name := c.QueryParam("name")
	subject := c.QueryParam("subject")
	message := c.QueryParam("message")

	ret := fmt.Sprintf("Post: Replying to %s: *%s* by *%s* with content *%s*\n", thread, subject, name, message)
	return c.String(http.StatusOK, ret)
}

func newThreadHandler(c echo.Context) (err error){
	name := c.FormValue("name")
	reply := c.FormValue("reply")
	subject := c.FormValue("subject")
	content := c.FormValue("message")

	file, err := c.FormFile("file")
	if err != nil{
		return err
	}
	src, err := file.Open()
	if err != nil{
		return err
	}
	defer src.Close()

	id := getId()
	ext := filepath.Ext(file.Filename)
	idstr := fmt.Sprintf("%08d%s", id, ext)

	dst, err := os.Create(idstr)
	if err != nil {
		return err
	}
	defer dst.Close()

	if _, err = io.Copy(dst, src); err != nil {
		return err
	}

	msg := Message{Subject: subject, Name: name, Content: content}
	// sanitze, make sure that reply id is a real id
	pid, err := strconv.ParseInt(reply, 10, 64)
	if err != nil {
		log.Fatal(pid)
	}
	if reply == "" {
		fmt.Printf("Thread\n")
		pid = id
		thread(id)
	} else {
		fmt.Printf("Post\n")
	}
	message(id, pid, msg)

	ret := fmt.Sprintf("Thread: Creating new thread no%d *%s* by *%s* with content *%s*\n", id, subject, name, content)
	return c.String(http.StatusOK, ret)
}
