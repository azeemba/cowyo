package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/boltdb/bolt"
)

type Watson struct {
	User        string
	Project     string
	Tags        []string
	DateTime    time.Duration
	AllTags     []string
	AllProjects []string
}

func getStatus(user string) Watson {
	tags := getItem(user, "tags")
	projects := getItem(user, "projects")
	db, err := bolt.Open("projects.db", 0600, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	currentProject := "No current project."
	currentTags := []string{""}
	currentTime := time.Since(time.Now())
	db.View(func(tx *bolt.Tx) error {
		// Assume bucket exists and has keys
		b := tx.Bucket([]byte(user))
		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			t1, e := time.Parse(time.RFC3339, string(k))
			if e == nil {
				currentTime = time.Since(t1)
				if string(v) == ">>stop<<" {
					currentProject = "None"
					currentTags = []string{""}
				} else {
					vals := strings.Split(string(v), ",")
					currentProject = vals[0]
					if len(vals) > 1 {
						currentTags = vals[1:len(vals)]
					}
				}
			} else {
				fmt.Println(e)
			}
		}
		return nil
	})
	return Watson{user, currentProject, currentTags, currentTime, tags, projects}
}

func addItem(user string, name string, itemType string) {
	db, err := bolt.Open("projects.db", 0600, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(user))
		return err
	})
	if err != nil {
		fmt.Errorf("create bucket: %s", err)
	}

	items := []string{}
	items = append(items, name)
	db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(user))
		v := b.Get([]byte(itemType))
		if v != nil {
			items = append(items, strings.Split(string(v), ",")...)
		}
		return nil
	})

	db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(user))
		err := b.Put([]byte(itemType), []byte(strings.Join(items, ",")))
		return err
	})
}

func startProject(user string, project string, tagString string) {

	db, err := bolt.Open("projects.db", 0600, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(user))
		return err
	})
	if err != nil {
		fmt.Errorf("create bucket: %s", err)
	}

	project = strings.TrimSpace(project)
	tagString = strings.TrimSpace(tagString)

	db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(user))
		err := b.Put([]byte(time.Now().Format(time.RFC3339)), []byte(project+","+tagString))
		return err
	})

}

func stopProject(user string) {

	db, err := bolt.Open("projects.db", 0600, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(user))
		return err
	})
	if err != nil {
		fmt.Errorf("create bucket: %s", err)
	}

	db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(user))
		err := b.Put([]byte(time.Now().Format(time.RFC3339)), []byte(">>stop<<"))
		return err
	})

}

func deleteItem(user string, name string, itemType string) {
	db, err := bolt.Open("projects.db", 0600, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(user))
		return err
	})
	if err != nil {
		fmt.Errorf("create bucket: %s", err)
	}

	items := []string{}
	db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(user))
		v := b.Get([]byte(itemType))
		if v != nil {
			items = strings.Split(string(v), ",")
		}
		return nil
	})

	j := 0
	for i := range items {
		j = i
		if items[i] == name {
			break
		}
	}
	items = append(items[:j], items[j+1:]...)

	db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(user))
		err := b.Put([]byte(itemType), []byte(strings.Join(items, ",")))
		return err
	})
}

func getItem(user string, itemType string) []string {
	projects := []string{}
	db, err := bolt.Open("projects.db", 0600, nil)
	if err != nil {
		log.Fatal(err)
	}

	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(user))
		return err
	})
	if err != nil {
		fmt.Errorf("create bucket: %s", err)
	}

	db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(user))
		v := b.Get([]byte(itemType))
		if v != nil {
			projects = strings.Split(string(v), ",")
		}
		return nil
	})
	db.Close()
	return projects
}
