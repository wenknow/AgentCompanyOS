package api

import (
	"net/http"

	"github.com/agentcompany/agent-company-os/backend/internal/app"
	"github.com/agentcompany/agent-company-os/backend/internal/database"
	"github.com/agentcompany/agent-company-os/backend/internal/model"
	redispkg "github.com/agentcompany/agent-company-os/backend/internal/redis"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	goredis "github.com/redis/go-redis/v9"
)

func NewRouter(services *app.Services, db *pgxpool.Pool, redisClient *goredis.Client) *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "database": database.Health(c.Request.Context(), db), "redis": redispkg.Health(c.Request.Context(), redisClient)})
	})
	v1 := r.Group("/api/v1")
	v1.GET("/runtime/status", func(c *gin.Context) {
		c.JSON(http.StatusOK, services.Runtime.Status())
	})
	v1.GET("/runtime/tools", func(c *gin.Context) {
		c.JSON(http.StatusOK, services.Runtime.Status().Tools)
	})
	v1.GET("/agents", func(c *gin.Context) {
		items, err := services.Agents.List(c.Request.Context())
		respond(c, items, err)
	})
	v1.GET("/projects", func(c *gin.Context) {
		items, err := services.Projects.List(c.Request.Context())
		respond(c, items, err)
	})
	v1.POST("/projects", func(c *gin.Context) {
		var req struct{ Name, Description, Owner string }
		if bind(c, &req) {
			return
		}
		p, err := services.CreateProject(c.Request.Context(), model.Project{Name: req.Name, Description: req.Description, Owner: req.Owner}, "api")
		respond(c, p, err)
	})
	v1.GET("/tasks", func(c *gin.Context) {
		items, err := services.Tasks.List(c.Request.Context(), 50)
		respond(c, items, err)
	})
	v1.POST("/tasks", func(c *gin.Context) {
		var req struct{ Agent, Title, CreatedBy string }
		if bind(c, &req) {
			return
		}
		if req.CreatedBy == "" {
			req.CreatedBy = "api"
		}
		res, err := services.Tasks.Assign(c.Request.Context(), req.Agent, req.Title, req.CreatedBy)
		respond(c, res, err)
	})
	v1.GET("/tasks/:id", func(c *gin.Context) {
		item, err := services.Tasks.Get(c.Request.Context(), c.Param("id"))
		if item == nil && err == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		respond(c, item, err)
	})
	v1.PATCH("/tasks/:id/status", func(c *gin.Context) {
		var req struct{ Status, Actor string }
		if bind(c, &req) {
			return
		}
		if req.Actor == "" {
			req.Actor = "api"
		}
		item, err := services.Tasks.UpdateStatus(c.Request.Context(), c.Param("id"), req.Status, req.Actor)
		respond(c, item, err)
	})
	v1.GET("/approvals", func(c *gin.Context) {
		items, err := services.Approvals.List(c.Request.Context(), c.Query("status"))
		respond(c, items, err)
	})
	v1.POST("/approvals", func(c *gin.Context) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "create approvals through task assignment in Phase 1"})
	})
	v1.POST("/approvals/:id/approve", func(c *gin.Context) {
		item, deployResult, err := services.ApproveApproval(c.Request.Context(), c.Param("id"), "api")
		respond(c, gin.H{"approval": item, "deployment": deployResult}, err)
	})
	v1.POST("/approvals/:id/reject", func(c *gin.Context) {
		var req struct{ Reason string }
		_ = c.ShouldBindJSON(&req)
		item, err := services.Approvals.Reject(c.Request.Context(), c.Param("id"), "api", req.Reason)
		respond(c, item, err)
	})

	v1.POST("/workflows/plan", func(c *gin.Context) {
		var req struct{ Idea, Actor string }
		if bind(c, &req) {
			return
		}
		if req.Actor == "" {
			req.Actor = "api"
		}
		res, err := services.Workflows.Plan(c.Request.Context(), req.Idea, req.Actor)
		respond(c, res, err)
	})
	v1.POST("/workflows/build", func(c *gin.Context) {
		var req struct{ Task, Actor string }
		if bind(c, &req) {
			return
		}
		if req.Actor == "" {
			req.Actor = "api"
		}
		res, err := services.Workflows.Build(c.Request.Context(), req.Task, req.Actor)
		respond(c, res, err)
	})
	v1.POST("/workflows/launch", func(c *gin.Context) {
		var req struct{ Topic, Actor string }
		if bind(c, &req) {
			return
		}
		if req.Actor == "" {
			req.Actor = "api"
		}
		res, err := services.Workflows.Launch(c.Request.Context(), req.Topic, req.Actor)
		respond(c, res, err)
	})
	v1.POST("/workflows/review", func(c *gin.Context) {
		var req struct{ Item, Actor string }
		if bind(c, &req) {
			return
		}
		if req.Actor == "" {
			req.Actor = "api"
		}
		res, err := services.Workflows.Review(c.Request.Context(), req.Item, req.Actor)
		respond(c, res, err)
	})
	v1.GET("/agent-runs", func(c *gin.Context) {
		items, err := services.TaskRepo.ListAgentRuns(c.Request.Context(), c.Query("task_id"), 50)
		respond(c, items, err)
	})
	v1.GET("/artifacts", func(c *gin.Context) {
		items, err := services.Artifacts.List(c.Request.Context(), c.Query("task_id"), 50)
		respond(c, items, err)
	})
	v1.GET("/artifacts/:id", func(c *gin.Context) {
		item, err := services.Artifacts.Get(c.Request.Context(), c.Param("id"))
		if item == nil && err == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		respond(c, item, err)
	})
	v1.GET("/reports/daily", func(c *gin.Context) {
		text, err := services.Reports.Daily(c.Request.Context())
		respond(c, gin.H{"report": text}, err)
	})
	v1.GET("/reports/weekly", func(c *gin.Context) {
		text, err := services.Reports.Weekly(c.Request.Context())
		respond(c, gin.H{"report": text}, err)
	})
	return r
}

func respond(c *gin.Context, data interface{}, err error) {
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "request failed"})
		return
	}
	c.JSON(http.StatusOK, data)
}

func bind(c *gin.Context, dst interface{}) bool {
	if err := c.ShouldBindJSON(dst); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return true
	}
	return false
}
