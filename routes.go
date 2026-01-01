package logger

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/gin-gonic/gin"
)

func RegisterRoute(group *gin.RouterGroup, method string, path string, handler gin.HandlerFunc, protected bool) {
	method = strings.ToUpper(method)

	group.Handle(method, path, handler)

	if !EnableLogs {
		return
	}

	icon := "→"
	if protected {
		icon = "🔒"
	}

	mColor := color.New(color.FgWhite, color.Bold)
	switch method {
	case "GET":
		mColor = color.New(color.FgBlue, color.Bold)
	case "POST":
		mColor = color.New(color.FgGreen, color.Bold)
	case "PUT":
		mColor = color.New(color.FgYellow, color.Bold)
	case "DELETE":
		mColor = color.New(color.FgRed, color.Bold)
	}

	fullPath := group.BasePath() + path

	fmt.Printf("%s %s %-16s %s\n",
		colorInfo.Sprint("[ROUTE]"),
		icon,
		mColor.Sprintf("[%s]", method),
		color.New(color.FgWhite).Sprint(fullPath),
	)
}
