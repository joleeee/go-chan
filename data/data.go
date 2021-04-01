package data

import (
	"github.com/xujiajun/nutsdb"
	"fmt"
	"log"
	"strconv"
	"bytes"
	"encoding/gob"
)

type MDB struct{
	db *nutsdb.DB
}

func New(db *nutsdb.DB) MDB{
	return MDB{db:db}
}

func (m *MDB) GetId() int64{
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
		log.Fatal(err)
	}

	fmt.Println("id is", id)
	return id
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
	Name string
	Subject string
	Content string
}

func (m *MDB) message(id int64, parent int64, msg Message){
	var network bytes.Buffer
	enc := gob.NewEncoder(&network)
	err := enc.Encode(msg)
	if err != nil {
		log.Fatal(err)
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
		log.Fatal(err)
	}
	// add post as child
	if err := m.db.Update(
		func(tx *nutsdb.Tx) error {
			bucket := "children"
			key := []byte(fmt.Sprintf("thread_%d", parent))
			val := []byte(fmt.Sprintf("%d", id))
			return tx.RPush(bucket, key, val)
	}); err != nil {
		log.Fatal(err)
	}
}

func (m *MDB) NewPost(id int64, pid int64, msg Message){
	m.message(id, pid, msg)
	if id == pid {
		m.thread(id)
	}
}

func (m *MDB) GetPost(id int64) Message {
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
				log.Fatal(err)
			}
		}
		return nil
	}); err != nil {
		log.Fatal(err)
	}
	return msg
}

func (m *MDB) thread(id int64){
	fmt.Printf("add %d to threadlist\n", id)
	if err := m.db.Update(
		func(tx *nutsdb.Tx) error {
			bucket := "threads"
			key := []byte("threadlist")
			val := []byte(fmt.Sprintf("%d", id))
			return tx.RPush(bucket, key, val)
	}); err != nil {
		log.Fatal(err)
	}
}

func (m *MDB) GetThreads() []string {
	var entr [][]byte
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
			log.Fatal(err)
		}
	var list []string
	for _, e := range entr {
		list = append(list, string(e))
	}
	return list
}

func (m *MDB) GetThreadPosts(thread string) ([]string, error) {
	var entr [][]byte
	var list []string
	if err := m.db.View(
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
			return list, err
		}
	for _, e := range entr {
		list = append(list, string(e))
	}
	return list, nil
}
