package data

import (
	"os"
	"io/ioutil"
	"fmt"
	"strconv"
	"path/filepath"
	"bytes"
	"encoding/gob"
)

type FS struct {
	path string
}

//func NewFS(name string) Database {
func NewFS(name string) FS {
	fs := FS{path:name}
	return fs
	//var iface Database = &fs
	//return iface
}

// GetId gets a unique id for a future post, 0..N
func (fs *FS) GetId() (int64, error) {
	path := fmt.Sprintf("%s/id", fs.path)
	var now int64 = 0
	if _, err := os.Stat(path); err == nil {
		// file exists
		f, err := os.OpenFile(path, os.O_RDWR, 0644)
		defer f.Close()
		if err != nil { return -1, err }

		byt, err := ioutil.ReadAll(f)
		if err != nil { return -1, err }
		str := string(byt)

		was, err := strconv.ParseInt(str, 16, 64)
		if err != nil { return -1, err }

		now = was + 1
	}

	err := os.MkdirAll(filepath.Dir(path), 0777)
	f, err := os.Create(path)
	defer f.Close()
	if err != nil {
		panic(err)
	}

	out := fmt.Sprintf("%x", now)
	d := []byte(out)
	_, err = f.Write(d)
	if err != nil {
		panic(err)
	}

	return now, nil
}

func (fs *FS) setThread(id int64, msg Message) error {
	path := fmt.Sprintf("%s/threadlist", fs.path)
	var threadlist []int64
	if _, err := os.Stat(path); err == nil {
		// file exists
		f, err := os.OpenFile(path, os.O_RDWR, 0644)
		defer f.Close()
		if err != nil { return err }

		byt, err := ioutil.ReadAll(f)
		if err != nil { return err }
		network := bytes.NewBuffer(byt)
		dec := gob.NewDecoder(network)
		err = dec.Decode(&threadlist)
		if err != nil { return err }
	}

	threadlist = append(threadlist, id)
	// save threadlist
	var network bytes.Buffer
	enc := gob.NewEncoder(&network)
	err := enc.Encode(threadlist)
	if err != nil { return err }

	f, err := os.Create(path)
	defer f.Close()
	if err != nil { return err }

	_, err = f.Write(network.Bytes())
	if err != nil { return err }

	return nil
}

func (fs *FS) GetThreads() ([]int64, error) {
	path := fmt.Sprintf("%s/threadlist", fs.path)
	var threadlist []int64

	f, err := os.OpenFile(path, os.O_RDWR, 0644)
	defer f.Close()
	if err != nil { return threadlist, err }

	byt, err := ioutil.ReadAll(f)
	if err != nil { return threadlist, err }
	network := bytes.NewBuffer(byt)
	dec := gob.NewDecoder(network)
	err = dec.Decode(&threadlist)
	if err != nil { return threadlist, err }

	return threadlist, nil
}

// writePost writs a post to disk
func (fs *FS) writePost(id int64, parent int64, msg Message) error {
	var network bytes.Buffer
	enc := gob.NewEncoder(&network)
	err := enc.Encode(msg)
	if err != nil { return err }

	path := fmt.Sprintf("%s/posts/%d", fs.path, id)
	err = os.MkdirAll(filepath.Dir(path), 0777)
	if err != err { return err }

	f, err := os.Create(path)
	defer f.Close()
	if err != nil { return err  }

	_, err = f.Write(network.Bytes())
	if err != nil { return err }

	return nil
}

// NewPost creates a new post OR THREAD
func (fs *FS) NewPost(id int64, parent int64, msg Message) error {
	err := fs.writePost(id, parent, msg)
	if err != nil { return err }

	// when id==parent, it's a thread
	if id == parent {
		err := fs.setThread(id, msg)
		fmt.Println(err)
		if err != nil { return err }
	}
	return nil
}

func (fs *FS) GetPost(id int64) (Message, error) {
	var msg Message

	path := fmt.Sprintf("%s/posts/%d", fs.path, id)
	f, err := os.OpenFile(path, os.O_RDWR, 0644)
	defer f.Close()
	if err != nil { return msg, err }

	byt, err := ioutil.ReadAll(f)
	if err != nil { return msg, err }

	network := bytes.NewBuffer(byt)
	dec := gob.NewDecoder(network)
	err = dec.Decode(&msg)
	if err != nil { return msg, err }

	return msg, nil
}
