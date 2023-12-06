package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

	var db *gorm.DB
	var validate *validator.Validate

	type (
		TodoModel struct {
			ID        uint      `json:"id" gorm:"primaryKey"`
			Title     string    `json:"title" validate:"required"`
			Completed bool      `json:"completed"`
			CreatedAt time.Time `json:"created_at"`
		}

		Todo struct {
			ID        uint      `json:"id"`
			Title     string    `json:"title" validate:"required"`
			Completed bool      `json:"completed"`
			CreatedAt time.Time `json:"created_at"`
		}
	)

	func init() {
		var err error

		// Initialize Gorm with MySQL
		dsn := "root:@tcp(127.0.0.1:3306)/final-exam?charset=utf8mb4&parseTime=True&loc=Local"
		db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
		if err != nil {
			log.Fatalf("Failed to connect to MySQL: %v", err)
		}

		// Migrate the database schema
		db.AutoMigrate(&TodoModel{})

		// Initialize validator
		validate = validator.New()
	}

	func HomeHandler(c *gin.Context) {
		c.HTML(http.StatusOK, "home.tpl", nil)
	}

	func CreateTodo(c *gin.Context) {
		var t Todo

		if err := c.ShouldBindJSON(&t); err != nil {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
			return
		}

		// Validate input
		if err := validate.Struct(t); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Create a todo
		tm := TodoModel{
			Title:     t.Title,
			Completed: false,
			CreatedAt: time.Now(),
		}

		db.Create(&tm)

		c.JSON(http.StatusCreated, gin.H{
			"message": "Todo created successfully",
			"todo_id": tm.ID,
		})
	}

	func UpdateTodo(c *gin.Context) {
		id := c.Param("id")
		var t Todo

		if err := c.ShouldBindJSON(&t); err != nil {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
			return
		}

		// Validate input
		if err := validate.Struct(t); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Update a todo
		db.Model(&TodoModel{}).Where("id = ?", id).Updates(map[string]interface{}{
			"title":     t.Title,
			"completed": t.Completed,
		})

		c.JSON(http.StatusOK, gin.H{
			"message": "Todo updated successfully",
		})
	}

	func FetchTodos(c *gin.Context) {
		var todos []TodoModel

		db.Find(&todos)

		todoList := make([]Todo, len(todos))
		for i, t := range todos {
			todoList[i] = Todo{
				ID:        t.ID,
				Title:     t.Title,
				Completed: t.Completed,
				CreatedAt: t.CreatedAt,
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"data": todoList,
		})
	}

	func DeleteTodo(c *gin.Context) {
		id := c.Param("id")

		db.Delete(&TodoModel{}, id)
		c.JSON(http.StatusOK, gin.H{
			"message": "Todo deleted successfully",
		})
	}

	func main() {
		stopChan := make(chan os.Signal)
		signal.Notify(stopChan, os.Interrupt)

		router := gin.Default()

		router.LoadHTMLGlob("templates/*.tpl")
		router.GET("/", HomeHandler)

		todoGroup := router.Group("/todo")
		{
			todoGroup.POST("/", CreateTodo)
			todoGroup.PUT("/:id", UpdateTodo)
			todoGroup.GET("/:id", FetchTodos)
			todoGroup.DELETE("/:id", DeleteTodo)
		}

		srv := &http.Server{
			Addr:         ":9000",
			Handler:      router,
			ReadTimeout:  60 * time.Second,
			WriteTimeout: 60 * time.Second,
			IdleTimeout:  60 * time.Second,
		}

		go func() {
			log.Println("Listening on port :9000")
			if err := srv.ListenAndServe(); err != nil {
				log.Printf("listen: %s\n", err)
			}
		}()

		<-stopChan
		log.Println("Shutting down server...")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		srv.Shutdown(ctx)
		defer cancel()
		log.Println("Server gracefully stopped!")
	}
