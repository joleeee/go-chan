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
	"html/template"
	"bytes"
)

var db *nutsdb.DB
var mdb data.MDB
func Init(datab *nutsdb.DB){
	mdb = data.New(datab)
}

func Root(c echo.Context) (err error){
	return c.String(http.StatusOK, "hello");
}

func ThreadList(c echo.Context) (err error){
	format := c.QueryParam("format")

	if format == "" {
		oot, err := mdb.GetThreads()
		if err != nil {
			return err
		}
		s := "threadlist<br>"
		for _, e := range oot {
			s2 := fmt.Sprintf("<a href=\"threads/%s\">%s</a><br>", e, e)
			s += s2
		}
		return c.HTML(http.StatusOK, s)
	} else if format == "raw" {
		return c.String(http.StatusOK, "rawr")
	} else {
		return c.String(http.StatusBadRequest, "format unrecognized\n")
	}
}

func Thread(c echo.Context) (err error){
	nr := c.Param("data")

	a, err := mdb.GetThreadPosts(nr)
	if err != nil {
		return c.String(http.StatusNotFound, "not found")
	}

	s := ""
	for _, e := range a{
		id, err := strconv.ParseInt(e, 10, 64)
		if err != nil {
			return err
		}
		msg, err := mdb.GetPost(id)
		if err != nil{
			return err
		}
		var post bytes.Buffer
		templ, _ := template.ParseFiles("templates/post.html")
		templ.Execute(&post, msg)

		s += post.String()
	}

	var thread bytes.Buffer
	templ, _ := template.ParseFiles("templates/thread.html")
	templ.Execute(&thread, template.HTML(s))

	return c.HTML(http.StatusOK, thread.String())
}

func NewPost(c echo.Context) (err error){
	name := c.FormValue("name")
	reply := c.FormValue("reply")
	subject := c.FormValue("subject")
	content := c.FormValue("message")

	// get file
	file, err := c.FormFile("file")
	if err != nil{
		return err
	}
	src, err := file.Open()
	if err != nil{
		return err
	}
	defer src.Close()

	// get id (don't get id before we know we can read file)
	id, err := mdb.GetId()
	if err != nil {
		return err
	}
	ext := filepath.Ext(file.Filename)
	idstr := fmt.Sprintf("%08d%s", id, ext)

	// write file
	dst, err := os.Create(idstr)
	if err != nil {
		return err
	}
	defer dst.Close()

	if _, err = io.Copy(dst, src); err != nil {
		return err
	}

	fmt.Println("a")
	msg := data.Message{Subject: subject, Name: name, Content: content}
	// sanitze, make sure that reply id is a real id
	pid, err := strconv.ParseInt(reply, 10, 64)
	if err != nil {
		pid = id
	}
	mdb.NewPost(id,pid,msg)

	ret := fmt.Sprintf("Thread: Creating new thread no%d *%s* by *%s* with content *%s*\n", id, subject, name, content)
	return c.String(http.StatusOK, ret)
}
