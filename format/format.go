package format

import (
	"fmt"
	"github.com/joleeee/go-chan/data"
	"bufio"
	"strings"
	"strconv"
	"html/template"
	"bytes"
	"path/filepath"
)

func FormatPost(rmsg *data.Message) string {
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
				// msg.ParentId doesn't work with crossposting...
				out += fmt.Sprintf(`<a href="%d#p%d">%s</a>`, msg.ParentId, id, text)
				continue
			}
		}
		if strings.HasPrefix(text, "&gt;") {
			out += fmt.Sprintf(`<span class="greentext">%s</span>`, text)
			continue
		}
		if strings.HasPrefix(text, "&lt;") {
			out += fmt.Sprintf(`<span class="pinktext">%s</span>`, text)
			continue
		}
		out += text
	}
	msg.Content = template.HTML(out)

	tf := template.FuncMap{
		"isImg": func(i interface{}) bool {
			if str, ok := i.(string); ok {
				ext := filepath.Ext(str)
				return ext==".jpg" || ext==".jpeg" || ext==".png" || ext==".webp"
			}
			return false
		},
		"isVid": func(i interface{}) bool {
			if str, ok := i.(string); ok {
				ext := filepath.Ext(str)
				return ext==".webm" || ext==".mp4" || ext==".mkv"
			}
			return false
		},
	}

	templ, _ := template.New("post.html").Funcs(tf).ParseFiles("templates/post.html")

	var post bytes.Buffer
	templ.Execute(&post, msg)

	return post.String()
}
