package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
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

var linkMap = map[string]*models.Link{
	"example": {Id: "example", Url: "https://example.com"},
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

	return views.Submission(id).Render(c.Request().Context(), c.Response())
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
	_, err := url.ParseRequestURI(s)
	return err == nil
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
