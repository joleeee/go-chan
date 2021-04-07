package tests

import (
	"testing"
	"os"

	"github.com/joleeee/go-chan/data"
)


func del() {
	os.RemoveAll("testdb/")
}

func TestID(t *testing.T) {
	del()
	fs := data.NewFS("testdb")
	var i int64
	for i = 0; i < 1000; i++{
		v, err := fs.GetId()
		if err != nil || i != v {
			t.Fatalf("%d %d %s", i, v, err)
		}
	}
}

func MakePost(fs *data.FS, pid int64, name string, text string) int64 {
	id, err := fs.GetId()
	if err != nil { panic("") }
	if pid == -1 { pid = id }

	msg := data.Message{Id: id,
		ParentId: pid,
		Name: name,
		Subject: "",
		Content: text,
		Time: "",
		Url: ""}

	fs.NewPost(id, pid, msg)
	return id
}

func TestPost(t *testing.T) {
	del()
	fs := data.NewFS("testdb")
	MakePost(&fs, -1, "rootname", "i like this website!")
	MakePost(&fs, 0, "person", "i agree")
	MakePost(&fs, -1, "person", "me too")
	MakePost(&fs, 2, "person", "me too")

	rmsg, _ := fs.GetPost(0)
	correct := rmsg.Id == 0 &&
		rmsg.ParentId == rmsg.Id &&
		rmsg.Name == "rootname" &&
		rmsg.Subject == "" &&
		rmsg.Content == "i like this website!" &&
		rmsg.Time == "" &&
		rmsg.Url == ""
	if !correct { t.Error("ERROR") }

	threadlist, err := fs.GetThreads()
	ans := []int64{0, 2}
	if err != nil { t.Error(err) }
	if func()bool{
		for i, v:= range threadlist {
			if v != ans[i] { return true }}; return false }() {
		t.Error("threadlist wrong")
	}
}

func TestClean(t *testing.T) {
	del()
}
