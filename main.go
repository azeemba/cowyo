package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/boltdb/bolt"
	"github.com/gin-gonic/gin"
)

// AllowedIPs is a white/black list of
// IP addresses allowed to access awwkoala
var AllowedIPs = map[string]bool{
	"192.168.1.13": true,
	"192.168.1.12": true,
	"192.168.1.2":  true,
}

// RuntimeArgs contains all runtime
// arguments available
var RuntimeArgs struct {
	WikiName         string
	ExternalIP       string
	Port             string
	DatabaseLocation string
	ServerCRT        string
	ServerKey        string
	SourcePath       string
	AdminKey         string
	Socket           string
}
var VersionNum string

func main() {
	VersionNum = "0.9"
	// _, executableFile, _, _ := runtime.Caller(0) // get full path of this file
	cwd, _ := os.Getwd()
	databaseFile := path.Join(cwd, "data.db")
	flag.StringVar(&RuntimeArgs.Port, "p", ":8003", "port to bind")
	flag.StringVar(&RuntimeArgs.DatabaseLocation, "db", databaseFile, "location of database file")
	flag.StringVar(&RuntimeArgs.AdminKey, "a", RandStringBytesMaskImprSrc(50), "key to access admin priveleges")
	flag.StringVar(&RuntimeArgs.ServerCRT, "crt", "", "location of ssl crt")
	flag.StringVar(&RuntimeArgs.ServerKey, "key", "", "location of ssl key")
	flag.StringVar(&RuntimeArgs.WikiName, "w", "AwwKoala", "custom name for wiki")
	flag.CommandLine.Usage = func() {
		fmt.Println(`AwwKoala (version ` + VersionNum + `): A Websocket Wiki and Kind Of A List Application
run this to start the server and then visit localhost at the port you specify
(see parameters).
Example: 'awwkoala yourserver.com'
Example: 'awwkoala -p :8080 localhost:8080'
Example: 'awwkoala -db /var/lib/awwkoala/db.bolt localhost:8003'
Example: 'awwkoala -p :8080 -crt ssl/server.crt -key ssl/server.key localhost:8080'
Options:`)
		flag.CommandLine.PrintDefaults()
	}
	flag.Parse()
	RuntimeArgs.ExternalIP = flag.Arg(0)
	if RuntimeArgs.ExternalIP == "" {
		RuntimeArgs.ExternalIP = GetLocalIP() + RuntimeArgs.Port
	}
	RuntimeArgs.SourcePath = cwd
	Open(RuntimeArgs.DatabaseLocation)
	defer Close()

	// create programdata bucket
	err := db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("programdata"))
		if err != nil {
			return fmt.Errorf("create bucket: %s", err)
		}
		return err
	})
	if err != nil {
		panic(err)
	}

	// Default page
	aboutFile, _ := ioutil.ReadFile(path.Join(RuntimeArgs.SourcePath, "templates/aboutpage.md"))
	p := WikiData{"about", "", []string{}, []string{}}
	p.save(string(aboutFile))

	// var q WikiData
	// q.load("about")
	// fmt.Println(getImportantVersions(q))

	r := gin.Default()
	r.LoadHTMLGlob(path.Join(RuntimeArgs.SourcePath, "templates/*"))
	r.GET("/", newNote)
	r.HEAD("/", func(c *gin.Context) { c.Status(200) })
	r.GET("/:title", editNote)
	r.GET("/:title/*option", everythingElse)
	r.DELETE("/listitem", deleteListItem)
	r.DELETE("/deletepage", deletePage)
	r.POST("/start", func(c *gin.Context) {
		user := strings.TrimSpace(strings.ToLower(c.PostForm("user")))
		currentProject := strings.TrimSpace(strings.ToLower(c.PostForm("currentProject")))
		tagString := strings.TrimSpace(strings.ToLower(c.PostForm("tagString")))
		startProject(user, currentProject, tagString)
		c.Redirect(302, "/"+user+"/projects")
	})
	r.POST("/stop", func(c *gin.Context) {
		user := strings.TrimSpace(strings.ToLower(c.PostForm("user")))
		stopProject(user)
		c.Redirect(302, "/"+user+"/projects")
	})
	r.POST("/add", func(c *gin.Context) {
		itemName := strings.TrimSpace(strings.ToLower(c.PostForm("itemName")))
		user := strings.TrimSpace(strings.ToLower(c.PostForm("user")))
		itemType := strings.TrimSpace(strings.ToLower(c.PostForm("itemType")))
		fmt.Println(user, itemName, itemType)
		addItem(user, itemName, itemType)
		c.Redirect(302, "/"+user+"/projects")
	})
	r.POST("/delete", func(c *gin.Context) {
		itemName := strings.TrimSpace(strings.ToLower(c.PostForm("itemName")))
		user := strings.TrimSpace(strings.ToLower(c.PostForm("user")))
		itemType := strings.TrimSpace(strings.ToLower(c.PostForm("itemType")))
		fmt.Println(user, itemName, itemType)
		deleteItem(user, itemName, itemType)
		c.Redirect(302, "/"+user+"/projects")
	})
	if RuntimeArgs.ServerCRT != "" && RuntimeArgs.ServerKey != "" {
		RuntimeArgs.Socket = "wss"
		fmt.Println("--------------------------")
		fmt.Println("AwwKoala (version " + VersionNum + ") is up and running on https://" + RuntimeArgs.ExternalIP)
		fmt.Println("Admin key: " + RuntimeArgs.AdminKey)
		fmt.Println("--------------------------")
		r.RunTLS(RuntimeArgs.Port, RuntimeArgs.ServerCRT, RuntimeArgs.ServerKey)
	} else {
		RuntimeArgs.Socket = "ws"
		fmt.Println("--------------------------")
		fmt.Println("AwwKoala (version " + VersionNum + ") is up and running on http://" + RuntimeArgs.ExternalIP)
		fmt.Println("Admin key: " + RuntimeArgs.AdminKey)
		fmt.Println("--------------------------")
		r.Run(RuntimeArgs.Port)
	}
}
