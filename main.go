package main

import (
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/heroku/x/hmetrics/onload"
	"github.com/yakkun/totsuka-ps-bot/models"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jinzhu/gorm/dialects/mysql"
)

func main() {
	port := os.Getenv("PORT")

	if port == "" {
		log.Fatal("$PORT must be set")
	}

	// Prepare http router (gin)
	router := gin.New()
	router.Use(gin.Logger())

	// Config for statis root
	router.LoadHTMLGlob("templates/*.tmpl.html")
	router.Static("/static", "static")

	// db connection (gorm)
	db := ConnectDB()
	defer db.Close()
	MigrateDB(db)

	// GET: /
	router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.tmpl.html", nil)
	})

	// POST: /callback
	router.POST("/callback", func(c *gin.Context) {
		callback(c, db)
	})

	// GET: /result/:id
	// ゲーム(id=game_id)の現在の状況及び結果を表示する
	router.GET("/result/:id", func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			showErrorHTML(c, http.StatusInternalServerError, "strconv error")
			return
		}
		if id <= 0 {
			showErrorHTML(c, http.StatusBadRequest, "Need valid id")
			return
		}
		game := models.Game{}
		db.First(&game, id)
		if game.ID == 0 {
			showErrorHTML(c, http.StatusNotFound, "Not found")
			return
		}
		c.HTML(http.StatusOK, "result.tmpl.html", gin.H{
			"game": game,
		})
	})

	router.GET("/result/:id/json", func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			showErrorJSON(c, http.StatusInternalServerError, "strconv error")
			return
		}
		if id <= 0 {
			showErrorJSON(c, http.StatusBadRequest, "Need valid id")
			return
		}
		game := models.Game{}
		db.First(&game, id)
		if game.ID == 0 {
			showErrorJSON(c, http.StatusNotFound, "Not found")
			return
		}
		transactions := []models.Transaction{}
		db.Model(&game).Related(&transactions)
		type Result struct {
			ID        uint
			Amount    int
			IsBuyin   bool
			CreatedAt time.Time
			User      models.User
			Game      models.Game
		}
		var result []Result
		for _, t := range transactions {
			// FIXME: N+1
			u := models.User{}
			db.Model(&t).Related(&u)
			var r Result
			r.ID = t.ID
			r.Amount = t.Amount
			r.IsBuyin = t.IsBuyin
			r.CreatedAt = t.CreatedAt
			r.User = u
			r.Game = game
			result = append(result, r)
		}

		c.JSON(http.StatusOK, result)
	})

	router.Run(":" + port)
}

func checkRegexp(reg, str string) bool {
	return regexp.MustCompile(reg).Match([]byte(str))
}

func showErrorHTML(c *gin.Context, code int, message string) {
	c.HTML(code, "error.tmpl.html", gin.H{
		"code":    code,
		"message": message,
	})
}

func showErrorJSON(c *gin.Context, code int, message string) {
	c.JSON(code, gin.H{
		"code":    code,
		"message": message,
	})
}
