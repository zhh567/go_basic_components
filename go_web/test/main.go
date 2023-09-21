package main

import (
	"context"
	"fmt"
	"go_web"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"text/template"
	"time"

	"log/slog"
)

func middlewareForV2() go_web.HandlerFunc {
	return func(c *go_web.Context) {
		t := time.Now()
		c.Next()
		slog.Info(fmt.Sprintf("[%d] %s in %v for group v2", c.StatusCode, c.Req.RequestURI, time.Since(t)))
	}
}

func middleForAll() go_web.HandlerFunc {
	return func(c *go_web.Context) {
		c.Next()
		slog.Info(fmt.Sprintf("new request [%d] %s", c.StatusCode, c.Req.RequestURI))
	}
}
func middleRecory() go_web.HandlerFunc {
	return func(c *go_web.Context) {
		defer func() {
			if err := recover(); err != nil {
				msg := fmt.Sprintf("%s", err)
				slog.Error(trace(msg))
				c.String(http.StatusInternalServerError, "Internal Server Error")
			}
		}()

		c.Next()
	}
}

// print stack trace for debug
func trace(message string) string {
	var pcs [32]uintptr
	n := runtime.Callers(3, pcs[:]) // skip first 3 caller

	var str strings.Builder
	str.WriteString(message + "\nTraceback:")
	for _, pc := range pcs[:n] {
		fn := runtime.FuncForPC(pc)
		file, line := fn.FileLine(pc)
		str.WriteString(fmt.Sprintf("\n\t%s:%d", file, line))
	}
	return str.String()
}

func main() {
	e := go_web.New()

	e.Use(middleRecory())
	e.Use(middleForAll())

	e.SetFuncMap(template.FuncMap{
		"FormatAsDate": func(t time.Time) string {
			year, month, day := t.Date()
			return fmt.Sprintf("%d-%02d-%02d", year, month, day)
		},
	})
	e.LoadHTMLGlob("./templates/*")
	e.Static("/assets", "./assets")

	e.GET("/", func(c *go_web.Context) {
		c.HTML(http.StatusOK, "index.html", go_web.H{
			"title": "gee",
			"now":   time.Date(2023, 9, 18, 0, 0, 0, 0, time.UTC),
		})
	})
	e.GET("/hello/:name", func(c *go_web.Context) {
		c.String(http.StatusOK, "hello %s, you're at %s\n", c.Param("name"), c.Path)
	})
	e.GET("/world/*param", func(c *go_web.Context) {
		c.JSON(http.StatusOK, go_web.H{"param": c.Param("param")})
	})

	v2 := e.Group("/v2")
	v2.Use(middlewareForV2())
	{
		v2.GET("/hello/:name", func(c *go_web.Context) {
			time.Sleep(time.Millisecond * 10)
			c.String(http.StatusOK, "hello %s", c.Param("name"))
		})
	}

	e.GET("panic", func(c *go_web.Context) {
		panic("proactively panic")
	})

	server := http.Server{
		Addr:    ":8080",
		Handler: e,
	}
	go func() {
		if err := server.ListenAndServe(); err != nil {
			slog.Error(fmt.Sprintf("server.ListenAndServe() error: %v", err))
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGQUIT)
	<-stop

	slog.Info("shutdown server ...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		slog.Error(fmt.Sprintf("server.Shutdown() error: %v", err))
	}
	slog.Info("server exiting")
}
