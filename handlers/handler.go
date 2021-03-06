package handlers

import (
	"os"
	"io"
	"path/filepath"
	"net/http"
	"fmt"
	"strconv"
	"html/template"
	"bytes"
	"time"

	"github.com/labstack/echo"

	"github.com/joleeee/go-chan/format"
	"github.com/joleeee/go-chan/data"
)

var mdb data.Database
func Init(name string){
	mdb = data.NewNuts(name)
}

func Root(c echo.Context) (err error){
	s := "<h2>NOchan - the front page of the shitternet</h2>"

	var body bytes.Buffer
	tempb, _ := template.ParseFiles("templates/body.html")
	tempb.Execute(&body, template.HTML(s))

	return c.HTML(http.StatusOK, body.String())
}

func ThreadList(c echo.Context) (err error){
	fmat := c.QueryParam("format")

	if fmat == "" {
		oot, err := mdb.GetThreads()
		if err != nil {
			return err
		}
		s := "<h2>Threadlist</h2>"

		// submit
		pArgs := postArgs{Id:"", Hide:true}
		var sub bytes.Buffer
		temps, _ := template.ParseFiles("templates/upload.html")
		temps.Execute(&sub, pArgs)
		s += sub.String()

		for _, id := range oot {
			rmsg, err := mdb.GetPost(id)
			if err != nil {
				continue
			}

			emsg := format.FormatPost(&rmsg)

			s += emsg
		}

		var body bytes.Buffer
		tempb, _ := template.ParseFiles("templates/body.html")
		tempb.Execute(&body, template.HTML(s))

		return c.HTML(http.StatusOK, body.String())
	} else if fmat == "raw" {
		return c.String(http.StatusOK, "rawr")
	} else {
		return c.String(http.StatusBadRequest, "fmat unrecognized\n")
	}
}

type postArgs struct {
	Id string
	Hide bool
}
type threadArgs struct {
	Id int64
	Content template.HTML
}
func Thread(c echo.Context) (err error){
	nr := c.Param("data")
	rootid, err := strconv.ParseInt(nr, 10, 64)
	if err != nil {
		// shouldn't happen
		return err
	}

	a, err := mdb.GetThreadPosts(rootid)
	if err != nil {
		return c.String(http.StatusNotFound, "not found")
	}

	posts := ""
	// submit
	pArgs := postArgs{Id:nr, Hide:true}
	var sub bytes.Buffer
	temps, _ := template.ParseFiles("templates/upload.html")
	temps.Execute(&sub, pArgs)
	posts += sub.String()

	// posts
	for _, id := range a{
		rmsg, err := mdb.GetPost(id)
		if err != nil {
			continue
		}

		emsg := format.FormatPost(&rmsg)
		posts += emsg
	}

	var thread bytes.Buffer
	tempt, _ := template.ParseFiles("templates/thread.html")
	tempt.Execute(&thread, threadArgs{Id:rootid, Content:template.HTML(posts)})

	var body bytes.Buffer
	tempb, _ := template.ParseFiles("templates/body.html")
	tempb.Execute(&body, template.HTML(thread.String()))

	return c.HTML(http.StatusOK, body.String())
}

func writeFile(c echo.Context, id int64, filename string) (string, error){
	// get file
	file, err := c.FormFile(filename)
	if err != nil{
		return "", err
	}
	src, err := file.Open()
	if err != nil{
		return "", err
	}
	defer src.Close()

	ext := filepath.Ext(file.Filename)
	// we can actually just change this later because
	// this not only saves the file, but we also store
	// where it is saved on the individual Messages
	idstr := fmt.Sprintf("img/%08d%s", id, ext)

	// write file
	dst, err := os.Create(idstr)
	if err != nil {
		return "", err
	}
	defer dst.Close()

	if _, err = io.Copy(dst, src); err != nil {
		return "", err
	}

	return "/"+idstr, nil
}

func NewPost(c echo.Context) (err error){
	name := c.FormValue("name")
	reply := c.FormValue("reply")
	subject := c.FormValue("subject")
	content := c.FormValue("message")

	// get id (would be better after checking file...)
	id, err := mdb.GetId()
	if err != nil {
		return err
	}

	// get reply id
	pid, err := strconv.ParseInt(reply, 10, 64)
	if err != nil {
		pid = id // it's a thread
	} else {
		// it's not a thread, make sure the parent is legit
		_, err = mdb.GetThreadPosts(pid)
		if err != nil {
			return c.String(http.StatusBadRequest, "you can't reply to a nonexistent thread")
		}
	}

	// get upload
	url, err := writeFile(c, id, "file")
	// threads require upload
	if pid == id && err != nil {
		return c.String(http.StatusBadRequest, "threads require an image")
	}

	if name == "" {
		name = "Ola"
	}

	if content == "" {
		return c.String(http.StatusBadRequest, "you need to say something!")
	}

	tstr := time.Now().Format("2006-01-02 15:04:05 -0700 MST")
	msg := data.Message{Id: id, ParentId: pid, Subject: subject, Name: name, Content: content, Time: tstr, Url: url}
	mdb.NewPost(id,pid,msg)

	//ret := fmt.Sprintf("Thread: Creating new thread no%d *%s* by *%s* with content *%s*\n", id, subject, name, content)
	ref := c.Request().Referer()
	red := fmt.Sprintf("%s#p%d", ref, id)
	return c.Redirect(http.StatusSeeOther, red)
	//return c.String(http.StatusOK, ret)
}
