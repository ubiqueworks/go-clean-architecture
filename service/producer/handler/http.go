package handler

import (
	"net/http"

	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/ubiqueworks/go-clean-architecture/framework"
	"github.com/ubiqueworks/go-clean-architecture/framework/component/transport/http"
	"github.com/ubiqueworks/go-clean-architecture/service/producer/domain"
)

type httpPublishRequest struct {
	Name    string `json:"name,omitempty"`
	Message string `json:"message,omitempty"`
}

type httpMessage struct {
	Name      string    `json:"name,omitempty"`
	Message   string    `json:"message,omitempty"`
	CreatedAt time.Time `json:"createdAt,omitempty"`
}

func InitHttpFunc(service framework.Service, _ framework.Component, router *gin.Engine) error {
	handler := service.Handler().(*serviceHandler)

	router.GET("/messages", getMessagesHandler(handler))
	router.POST("/publish", publishHandler(handler))

	return nil
}

func getMessagesHandler(handler *serviceHandler) gin.HandlerFunc {
	return func(c *gin.Context) {
		logger := microhttp.RequestLogger(c)

		messageEntities, err := handler.getMessages(logger)
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		messages := make([]*httpMessage, 0)

		for _, m := range messageEntities {
			msg := &httpMessage{
				Name:      m.Name,
				Message:   m.Message,
				CreatedAt: m.CreatedAt,
			}
			messages = append(messages, msg)
		}
		c.JSON(http.StatusOK, messages)
	}
}

func publishHandler(handler *serviceHandler) gin.HandlerFunc {
	return func(c *gin.Context) {
		logger := microhttp.RequestLogger(c)
		requestId := microhttp.RequestId(c)

		var body httpPublishRequest
		if err := c.ShouldBindWith(&body, binding.JSON); err != nil {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}

		if body.Name == "" || body.Message == "" {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}

		message := domain.NewMessage(body.Name, body.Message)
		logger.Debug().Msgf("%v", message)

		if err := handler.storeAndPublishMessage(logger, requestId, message); err != nil {
			logger.Error().Err(err).Msg("server error")
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		c.Status(http.StatusCreated)
	}
}
