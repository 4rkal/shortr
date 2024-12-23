package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/4rkal/shortr/models"
	"github.com/4rkal/shortr/views"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	_ "github.com/lib/pq"
)

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

type FormData struct {
	Url string `form:"url"`
}

type StatsFormData struct {
	Id string `form:"id"`
}

var linkMap = map[string]*models.Link{}
var linkMapMutex sync.RWMutex

var baseurl *string
var db *sql.DB

func init() {
	baseurl = flag.String("url", "127.0.0.1:8080", "The URL (domain) that the server is running on")
	flag.Parse()

	var err error
	db, err = sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		panic(fmt.Sprintf("Failed to connect to database: %v", err))
	}

	if err := loadLinksIntoMemory(); err != nil {
		panic(fmt.Sprintf("Failed to load links into memory: %v", err))
	}

	_, err = db.Exec(`
	    CREATE TABLE IF NOT EXISTS links (
	        id VARCHAR(255) PRIMARY KEY,
	        url TEXT NOT NULL,
	        clicks INTEGER NOT NULL DEFAULT 0
	    )
	`)
	if err != nil {
		log.Fatal("Error creating table: ", err)
	}
}

func main() {
	e := echo.New()

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.Secure())

	e.GET("/stats", StatsHandler)
	e.GET("/:id", RedirectHandler)
	e.GET("/:id/", RedirectHandler)
	e.GET("/", IndexHandler)
	e.POST("/submit", SubmitHandler)
	e.GET("/health", HealthHandler)

	e.Logger.Fatal(e.Start(":8080"))
}

func RedirectHandler(c echo.Context) error {
	id := c.Param("id")

	linkMapMutex.RLock()
	link, found := linkMap[id]
	linkMapMutex.RUnlock()
	if !found {
		return c.String(http.StatusNotFound, "Link not found")
	}

	if !strings.Contains(link.Url, "://") {
		link.Url = "http://" + link.Url
	}

	link.Clicks = link.Clicks + 1

	go func(id string, clicks int) {
		_, err := db.Exec("UPDATE links SET clicks = $1 WHERE id = $2", clicks, id)
		if err != nil {
			fmt.Printf("Failed to update clicks for id %s: %v\n", id, err)
		}
	}(id, link.Clicks)

	return c.Redirect(http.StatusMovedPermanently, link.Url)
}

func IndexHandler(c echo.Context) error {
	return views.Index().Render(c.Request().Context(), c.Response())
}

func SubmitHandler(c echo.Context) error {
	var data FormData
	if err := c.Bind(&data); err != nil {
		return err
	}
	fmt.Println(data)

	if !isURL(data.Url) {
		return c.JSON(http.StatusBadRequest, "not a valid url")
	}

	var id string
	for {
		id = generateRandomString(6)
		if _, exists := linkMap[id]; !exists {
			break
		}
	}

	linkMapMutex.Lock()
	linkMap[id] = &models.Link{Id: id, Url: data.Url}
	linkMapMutex.Unlock()

	_, err := db.Exec("INSERT INTO links (id, url, clicks) VALUES ($1, $2, $3)", id, data.Url, 0)
	if err != nil {
		return fmt.Errorf("failed to save link to database: %v", err)
	}

	return views.Submission(id, *baseurl).Render(c.Request().Context(), c.Response())
}

func StatsHandler(c echo.Context) error {
	id := c.QueryParam("id")
	if id == "" {
		return views.StatsForm().Render(c.Request().Context(), c.Response())
	} else {
		link, found := linkMap[id]
		if !found {
			return c.String(http.StatusNotFound, "Id not found")
		}

		return views.Stats(link).Render(c.Request().Context(), c.Response())
	}
}

func HealthHandler(c echo.Context) error {
	return c.JSON(http.StatusOK, "ok")
}

func isURL(s string) bool {
	if !strings.Contains(s, "://") {
		s = "http://" + s
	}

	parsedUrl, err := url.ParseRequestURI(s)
	if err != nil {
		return false
	}

	if parsedUrl.Host == "" {
		return false
	}

	domainRegex := `^([a-zA-Z0-9-]+\.)+[a-zA-Z]{2,}$`
	matched, err := regexp.MatchString(domainRegex, parsedUrl.Host)
	if err != nil || !matched {
		return false
	}

	return true
}

func generateRandomString(length int) string {
	seededRand := rand.New(rand.NewSource(time.Now().UnixNano()))
	var result []byte
	for i := 0; i < length; i++ {
		index := seededRand.Intn(len(charset))
		result = append(result, charset[index])
	}
	return string(result)
}

func loadLinksIntoMemory() error {
	rows, err := db.Query("SELECT id, url, clicks FROM links")
	if err != nil {
		return fmt.Errorf("query error: %v", err)
	}
	defer rows.Close()

	linkMapMutex.Lock()
	defer linkMapMutex.Unlock()

	for rows.Next() {
		var id, url string
		var clicks int
		if err := rows.Scan(&id, &url, &clicks); err != nil {
			return fmt.Errorf("scan error: %v", err)
		}
		linkMap[id] = &models.Link{Id: id, Url: url, Clicks: clicks}
	}
	return nil
}
