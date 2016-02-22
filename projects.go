package main

import (
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/boltdb/bolt"
)

type Watson struct {
	User        string
	Project     string
	Tags        []string
	Duration    string
	AllTags     []string
	AllProjects []string
}

func addToTable(s string, currentDay string, projectBreakdown map[string]int, projectTagBreakdown map[string]map[string]int) string {
	n := map[int][]string{}
	var a []int
	for k, v := range projectBreakdown {
		n[v] = append(n[v], k)
	}
	for k := range n {
		a = append(a, k)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(a)))
	for _, v := range a {
		for _, k := range n[v] {
			s = s + "| " + currentDay
			timeString := (time.Duration(v) * time.Minute).String()
			timeString = timeString[0 : len(timeString)-2]
			s = s + "|" + k + "(" + timeString + ") | "
			if _, ok := projectTagBreakdown[k]; ok {
				for k2, v2 := range projectTagBreakdown[k] {
					timeString2 := (time.Duration(v2) * time.Minute).String()
					timeString2 = timeString[0 : len(timeString2)-2]
					s = s + k2 + "(" + timeString2 + ")  "
				}
			}
			s = s + "\n"
			currentDay = ""
		}
	}
	return s
}

func getReport(user string) string {
	db, err := bolt.Open("projects.db", 0600, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	markdownTable := `# Projects Report

| Day | Activity | Tags
| ----------------- | -------------- | -------------
`
	const longForm = "01/02/06"

	currentDay := ""
	projectBreakdown := make(map[string]int)
	projectTagBreakdown := make(map[string]map[string]int)
	currentProject := ""
	currentTags := []string{}
	startTime := time.Now()

	db.View(func(tx *bolt.Tx) error {
		// Assume bucket exists and has keys
		b := tx.Bucket([]byte(user))
		c := b.Cursor()

		for k, v := c.First(); k != nil; k, v = c.Next() {
			t1, e := time.Parse(time.RFC3339, string(k))
			if e == nil {
				// if its a different day, then reset everything
				if t1.Format(longForm) != currentDay {
					markdownTable = addToTable(markdownTable, currentDay, projectBreakdown, projectTagBreakdown)
					currentDay = t1.Format(longForm)
					projectBreakdown = make(map[string]int)
					projectTagBreakdown = make(map[string]map[string]int)
				}

				if string(v) == ">>stop<<" { // if we have a stop
					if currentProject != "" && t1.Sub(startTime).Minutes() > 1 {
						if val, ok := projectBreakdown[currentProject]; ok {
							projectBreakdown[currentProject] = val + int(t1.Sub(startTime).Minutes())
						} else {
							projectBreakdown[currentProject] = int(t1.Sub(startTime).Minutes())
							projectTagBreakdown[currentProject] = make(map[string]int)
						}
						for _, tag := range currentTags {
							if len(tag) > 2 {
								if val, ok := projectTagBreakdown[currentProject][tag]; ok {
									projectTagBreakdown[currentProject][tag] = val + int(t1.Sub(startTime).Minutes())
								} else {
									projectTagBreakdown[currentProject][tag] = int(t1.Sub(startTime).Minutes())
								}
							}
						}
					}
					currentProject = ""
				} else { // if we encounter another project
					vals := strings.Split(string(v), ",")
					newProject := vals[0]
					if currentProject != "" && newProject != currentProject && t1.Sub(startTime).Minutes() > 1 {
						if val, ok := projectBreakdown[currentProject]; ok {
							projectBreakdown[currentProject] = val + int(t1.Sub(startTime).Minutes())
						} else {
							projectBreakdown[currentProject] = int(t1.Sub(startTime).Minutes())
							projectTagBreakdown[currentProject] = make(map[string]int)
						}
						for _, tag := range currentTags {
							if len(tag) > 2 {
								if val, ok := projectTagBreakdown[currentProject][tag]; ok {
									projectTagBreakdown[currentProject][tag] = val + int(t1.Sub(startTime).Minutes())
								} else {
									projectTagBreakdown[currentProject][tag] = int(t1.Sub(startTime).Minutes())
								}
							}
						}
					}
					currentProject = newProject
					startTime = t1
					if len(vals) > 1 {
						currentTags = vals[1:len(vals)]
					}
				}
			} else {
				fmt.Println(e)
			}
		}
		markdownTable = addToTable(markdownTable, currentDay, projectBreakdown, projectTagBreakdown)
		// fmt.Println(markdownTable)
		return nil
	})
	return markdownTable
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

	duration := "a minute ago"
	if currentTime.Minutes() < 2 {
	} else if currentTime.Minutes() < 60 {
		mins := strconv.Itoa(int(currentTime.Minutes()))
		duration = mins + " minutes ago"
	} else {
		mins := strconv.Itoa(int(currentTime.Minutes()))
		hrs := strconv.Itoa(int(currentTime.Hours()))
		duration = hrs + "hr " + mins + "min ago"
	}

	return Watson{user, currentProject, currentTags, duration, tags, projects}
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
