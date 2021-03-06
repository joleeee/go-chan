package data

import (
	"fmt"
	"log"
	"strconv"
	"bytes"
	"encoding/gob"
	"html/template"
	"html"
	"sort"

	"github.com/xujiajun/nutsdb"
)

type Database interface {
	// initilizes db, creates if doesn't exist
	InitId() error

	// returns id that new post will be
	GetId() (int64, error)

	// creates new post
	NewPost(int64, int64, Message) error

	// returns post by id
	GetPost(int64) (Message, error)

	// returns list of thread ids
	//GetThreadIds() ([]int64, error)
	GetThreads() ([]int64, error)

	// returns list of ids of posts in thread
	GetThreadPosts(int64) ([]int64, error)
}

type MDB struct {
	db *nutsdb.DB
}

func NewNuts(name string) Database {
	opt := nutsdb.DefaultOptions
	opt.Dir = name + "-nuts"
	db, err := nutsdb.Open(opt)
	if err != nil {
		log.Fatal(err)
	}

	mdb := MDB{db:db}
	var iface Database = &mdb
	return iface
}

func (m *MDB) GetId() (int64, error) {
	var id int64

	// get id
	if err := m.db.View(
	func(tx *nutsdb.Tx) error{
		key := []byte("id")
		bucket := "internal"
		if e, err := tx.Get(bucket, key); err != nil {
			return err
		} else {
			str := string(e.Value)

			var ierr error
			id, ierr = strconv.ParseInt(str, 10, 64)
			if ierr != nil{
				return err
			}
		}
		return nil
	}); err != nil {
		return -1, err
	}

	//set id
	if err := m.db.Update(
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
		return -1, err
	}

	return id, nil
}

func (m *MDB) InitId() error {
	// get id
	err := m.db.View(
		func(tx *nutsdb.Tx) error{
			key := []byte("id")
			bucket := "internal"
			_, err := tx.Get(bucket, key)
			return err
		})

	// no error, id exists
	if err == nil {
		return nil
	}

	// error, set id=0
	fmt.Println("No id found, resetting it") // ask for confirmation?
	if err := m.db.Update(
		func(tx *nutsdb.Tx) error {
			key := []byte("id")
			str := fmt.Sprintf("%d", 0)
			val := []byte(str)
			bucket := "internal"
			if err := tx.Put(bucket, key, val, 0); err != nil {
				return err
			}
			return nil
	}); err != nil {
		log.Fatal(err)
	}

	return nil
}

type Message struct {
	Id int64
	ParentId int64
	Name string
	Subject string
	Content string
	Time string
	Url string
}

type EscapedMessage struct {
	Id int64
	ParentId int64
	Name string
	Subject string
	Content template.HTML
	Time string
	Url string
}

func (msg *Message) Escaped() EscapedMessage {
	esc := template.HTML(html.EscapeString(msg.Content))
	return EscapedMessage{Id:msg.Id,
		ParentId:msg.ParentId,
		Name:msg.Name,
		Subject:msg.Subject,
		Content:esc,
		Time:msg.Time,
		Url:msg.Url}
}

func (m *MDB) message(id int64, parent int64, msg Message) error{
	var network bytes.Buffer
	enc := gob.NewEncoder(&network)
	err := enc.Encode(msg)
	if err != nil {
		log.Print(err)
		return err
	}
	// add post
	if err := m.db.Update(
		func(tx *nutsdb.Tx) error {
			key := []byte(fmt.Sprintf("%d", id))
			val := network.Bytes()
			bucket := "posts"
			if err := tx.Put(bucket, key, val, 0); err != nil {
				return err
			}
			return nil
	}); err != nil {
		log.Print(err)
		return err
	}
	// add post as child
	if err := m.db.Update(
		func(tx *nutsdb.Tx) error {
			bucket := "children"
			key := []byte(fmt.Sprintf("thread_%d", parent))
			val := []byte(fmt.Sprintf("%d", id))
			return tx.RPush(bucket, key, val)
	}); err != nil {
		log.Print(err)
		return err
	}

	return nil
}

func (m *MDB) NewPost(id int64, pid int64, msg Message) error {
	err := m.message(id, pid, msg)
	if err != nil {
		return err
	}
	if id == pid { // threads are children of themselves
		err := m.thread(id)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *MDB) GetPost(id int64) (Message, error) {
	var msg Message
	if err := m.db.View(
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
				return err
			}
		}
		return nil
	}); err != nil {
		log.Print(err)
		return msg, err
	}
	return msg, nil
}

func (m *MDB) thread(id int64) error {
	fmt.Printf("add %d to threadlist\n", id)
	if err := m.db.Update(
		func(tx *nutsdb.Tx) error {
			bucket := "threads"
			key := []byte("threadlist")
			val := []byte(fmt.Sprintf("%d", id))
			return tx.RPush(bucket, key, val)
	}); err != nil {
		log.Print(err)
		return err
	}
	return nil
}

func (m *MDB) GetThreads() ([]int64, error) {
	var entr [][]byte
	var list []int64
	if err := m.db.View(
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
			log.Print(err)
			return list, err
		}
	for _, e := range entr {
		i, e := strconv.ParseInt(string(e), 10, 64)
		if e == nil {
			list = append(list, i)
		}
	}
	sort.Slice(list, func(i, j int) bool { return list[i] < list[j] })
	return list, nil
}

func (m *MDB) GetThreadPosts(thread int64) ([]int64, error) {
	var entr [][]byte
	var list []int64
	if err := m.db.View(
		func(tx *nutsdb.Tx) error {
			bucket := "children"
			key := []byte("thread_"+fmt.Sprintf("%d", thread))
			if items, err := tx.LRange(bucket, key, 0, -1); err != nil {
				return err
			} else {
				entr = items
			}
			return nil
		}); err != nil {
			return list, err
		}
	for _, e := range entr {
		i, e := strconv.ParseInt(string(e), 10, 64)
		if e == nil {
			list = append(list, i)
		}
	}
	return list, nil
}
