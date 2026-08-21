package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/langgenius/dify-sandbox/internal/service"
)

// RunCommandRequest is the wire format for POST /v1/sandbox/run/command.
//
// The endpoint is intentionally narrow: it accepts a command basename
// (e.g. "python3"), a list of arguments, and an optional work_dir that is
// resolved relative to upload_dir. A typical workflow is:
//
//	POST /v1/sandbox/file/upload        (uploads hello.py)
//	POST /v1/sandbox/run/command {      (runs python3 hello.py)
//	    "command": "python3",
//	    "args": ["hello.py"]
//	}
//
// Every field is independently validated by service.RunCommand before any
// process is spawned, so a malicious request is rejected up front rather
// than silently producing an unexpected error from the child process.
type RunCommandRequest struct {
	Command       string   `json:"command" form:"command" binding:"required"`
	Args          []string `json:"args" form:"args"`
	WorkDir       string   `json:"work_dir" form:"work_dir"`
	Timeout       int      `json:"timeout" form:"timeout"`
	EnableNetwork bool     `json:"enable_network" form:"enable_network"`
}

// RunCommandController handles POST /v1/sandbox/run/command requests.
//
// It re-uses the BindRequest helper used by the rest of the controllers
// so that bad JSON/form bodies get the same 400-shaped envelope as the
// other endpoints. The middleware stack (auth, max-workers, max-requests,
// trace) is applied in router.go.
func RunCommandController(c *gin.Context) {
	BindRequest(c, func(req RunCommandRequest) {
		c.JSON(http.StatusOK, service.RunCommand(c.Request.Context(), &service.CommandOptions{
			Command:       req.Command,
			Args:          req.Args,
			WorkDir:       req.WorkDir,
			Timeout:       req.Timeout,
			EnableNetwork: req.EnableNetwork,
		}))
	})
}
