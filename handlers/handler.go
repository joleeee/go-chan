package handlers

import (
	"os"
	"io"
	"path/filepath"
	"github.com/labstack/echo"
	"github.com/xujiajun/nutsdb"
	"net/http"
	"fmt"
	"strconv"
	"github.com/joleeee/go-chan/data"
	"log"
)

var db *nutsdb.DB
var mdb data.MDB
func Init(datab *nutsdb.DB){
	mdb = data.New(datab)
}

func Root(c echo.Context) (err error){
	return c.String(http.StatusOK, "hello");
}

func Posts(c echo.Context) (err error){
	format := c.QueryParam("format")

	if format == "" {
		oot := mdb.GetThreads()
		s := "threadlist<br>"
		for _, e := range oot {
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

func Post(c echo.Context) (err error){
	nr := c.Param("data")

	a := mdb.GetThreadPosts(nr)

	s := ""
	for _, e := range a{
		id, err := strconv.ParseInt(e, 10, 64)
		if err != nil {
			log.Fatal(err)
		}
		msg := mdb.GetPost(id)
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

func NewPost(c echo.Context) (err error){
	thread := c.QueryParam("thread")
	name := c.QueryParam("name")
	subject := c.QueryParam("subject")
	message := c.QueryParam("message")

	ret := fmt.Sprintf("Post: Replying to %s: *%s* by *%s* with content *%s*\n", thread, subject, name, message)
	return c.String(http.StatusOK, ret)
}

func NewThread(c echo.Context) (err error){
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

	id := mdb.GetId()
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

	msg := data.Message{Subject: subject, Name: name, Content: content}
	// sanitze, make sure that reply id is a real id
	pid, err := strconv.ParseInt(reply, 10, 64)
	if err != nil {
		log.Fatal(pid)
	}
	if reply == "" {
		pid = id
	}
	mdb.NewPost(id,pid,msg)

	ret := fmt.Sprintf("Thread: Creating new thread no%d *%s* by *%s* with content *%s*\n", id, subject, name, content)
	return c.String(http.StatusOK, ret)
}
