package main

import (
	"flag"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/4rkal/shortr/models"
	"github.com/4rkal/shortr/views"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

type FormData struct {
	Url string `form:"url"`
}

type StatsFormData struct {
	Id string `form:"id"`
}

var linkMap = map[string]*models.Link{}

var baseurl *string

func init() {
	baseurl = flag.String("url", "127.0.0.1:8080", "The url (domain) that the server is running on")

	flag.Parse()
}

func main() {
	e := echo.New()

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.Secure())

	e.GET("/stats", StatsHandler)
	e.POST("/stats", StatsSubmissionHandler)
	e.GET("/:id", RedirectHandler)
	e.GET("/", IndexHandler)
	e.POST("/submit", SubmitHandler)

	e.Logger.Fatal(e.Start(":8080"))
}

func RedirectHandler(c echo.Context) error {
	id := c.Param("id")

	link, found := linkMap[id]
	if !found {
		return c.String(http.StatusNotFound, "Link not found")
	}

	if !strings.Contains(link.Url, "://") {
		link.Url = "http://" + link.Url
	}

	link.Clicks = link.Clicks + 1

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

	linkMap[id] = &models.Link{Id: id, Url: data.Url}

	return views.Submission(id, *baseurl).Render(c.Request().Context(), c.Response())
}

func StatsHandler(c echo.Context) error {
	return views.StatsForm().Render(c.Request().Context(), c.Response())
}

func StatsSubmissionHandler(c echo.Context) error {
	var data StatsFormData
	if err := c.Bind(&data); err != nil {
		return err
	}

	link, found := linkMap[data.Id]
	if !found {
		return c.String(http.StatusNotFound, "Id not found")
	}

	return views.Stats(link).Render(c.Request().Context(), c.Response())
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
