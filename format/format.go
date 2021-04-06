package format

import (
	"fmt"
	"github.com/joleeee/go-chan/data"
	"bufio"
	"strings"
	"strconv"
	"html/template"
	"bytes"
)

func FormatPost(rmsg *data.Message, rootid int64) string {
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

	return post.String()
}
