package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/langgenius/dify-sandbox/internal/service"
	"github.com/langgenius/dify-sandbox/internal/types"
)

// UploadFileController handles file upload requests
func UploadFileController(c *gin.Context) {
	// Get the file from the request
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse(-400, "failed to get file from request"))
		return
	}

	// Open the file
	fileReader, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.ErrorResponse(-500, "failed to open file"))
		return
	}
	defer fileReader.Close()

	// Call the service to save the file
	resp, err := service.UploadFile(fileReader, file.Filename, file.Size)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.ErrorResponse(-500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, resp)
}

// DownloadFileController handles file download requests
func DownloadFileController(c *gin.Context) {
	BindRequest(c, func(req struct {
		Filename string `json:"filename" form:"filename" binding:"required"`
	}) {
		// Get the file path from the service
		filePath, err := service.DownloadFile(req.Filename)
		if err != nil {
			c.JSON(http.StatusNotFound, types.ErrorResponse(-404, err.Error()))
			return
		}

		// Send the file
		c.File(filePath)
	})
}
