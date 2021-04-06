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
	"time"
	"bufio"
	"strings"
)

var mdb data.MDB
func Init(datab *nutsdb.DB){
	mdb = data.New(datab)
}

func Root(c echo.Context) (err error){
	s := "<h2>NOchan - the front page of the shitternet</h2>"

	var body bytes.Buffer
	tempb, _ := template.ParseFiles("templates/body.html")
	tempb.Execute(&body, template.HTML(s))

	return c.HTML(http.StatusOK, body.String())
}

func ThreadList(c echo.Context) (err error){
	format := c.QueryParam("format")

	if format == "" {
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

		for _, e := range oot {
			id, err := strconv.ParseInt(e, 10, 64)
			if err != nil {
				// this should never happen!
				return err
			}
			msg, err := mdb.GetPost(id)

			var post bytes.Buffer
			templ, _ := template.ParseFiles("templates/post.html")
			templ.Execute(&post, msg)

			s += post.String()
		}

		var body bytes.Buffer
		tempb, _ := template.ParseFiles("templates/body.html")
		tempb.Execute(&body, template.HTML(s))

		return c.HTML(http.StatusOK, body.String())
	} else if format == "raw" {
		return c.String(http.StatusOK, "rawr")
	} else {
		return c.String(http.StatusBadRequest, "format unrecognized\n")
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

	a, err := mdb.GetThreadPosts(nr)
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
	for _, e := range a{
		id, err := strconv.ParseInt(e, 10, 64)
		if err != nil {
			return err
		}
		rmsg, err := mdb.GetPost(id)
		if err != nil {
			return err
		}
		msg := rmsg.Escaped()

		scan := bufio.NewScanner(strings.NewReader(string(msg.Content)))
		out := ""
		for scan.Scan() {
			if out != "" {
				out += "\n"
			}
			text := scan.Text()
			if strings.HasPrefix(text, "&gt;&gt;") {
				rest := text[len("&gt;&gt;"):]
				id, e := strconv.ParseInt(rest, 10, 64)
				if e == nil {
					out += fmt.Sprintf(`<a href="%d#p%d">%s</a>`, rootid, id, text)
					continue
				}
			}
			if strings.HasPrefix(text, "&gt;") {
				out += fmt.Sprintf(`<span class="greentext">%s</span>`, text)
				continue
			}
			out += text
		}
		msg.Content = template.HTML(out)

		var post bytes.Buffer
		templ, _ := template.ParseFiles("templates/post.html")

		templ.Execute(&post, msg)

		posts += post.String()
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
		_, err = mdb.GetThreadPosts(reply)
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
