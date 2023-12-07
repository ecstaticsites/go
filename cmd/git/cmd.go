package git

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"cbnr/util"

	"github.com/spf13/cobra"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/asim/git-http-backend/server"
)

// when you do "git push" to a remote, the first request is a GET to the
// location /reponame/info/refs. When we detect that happens, we init a bare
// repository at the requested location, so pushes always work
func GitInitBare(next http.Handler) http.Handler {
  return http.HandlerFunc(func(out http.ResponseWriter, req *http.Request) {

  	log.Printf("git init bare %s", "/tmp" + req.URL.Path)

  	cmd := exec.Command("git", "init", "--bare", "/tmp/aaaa")
    stdout, err := cmd.Output()

    if err != nil {
       log.Printf(err.Error())
       return
    }

    // Print the output
    log.Printf(string(stdout))

    next.ServeHTTP(out, req)
	})
}


var GitCmd = &cobra.Command{
	Use:   "git",
	Short: "git - handles administrative tasks regarding bunny and supabase",
	Run: func(cmd *cobra.Command, args []string) {

		log.Printf("STARTING")

		// set up channel to handle graceful shutdown
		done := make(chan os.Signal, 1)
		signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

		httpPort, err := util.GetEnvConfig("HTTP_LISTENER_PORT")
		if err != nil {
			log.Fatalf("Unable to get http listener port from environment: %v", err)
		}

		r := chi.NewRouter()

		r.Use(middleware.Recoverer)
		r.Use(middleware.Logger)
		r.Use(GitInitBare)
		//r.Use(middleware.AllowContentType("application/json"))
		r.Use(middleware.Timeout(time.Second))

		r.Get("/aaaa", server.Handler())
		r.Get("/aaaa/*", server.Handler())
		r.Post("/aaaa", server.Handler())
		r.Post("/aaaa/*", server.Handler())

		log.Printf("MIDDLEWARES SET UP, WILL LISTEN ON %v...", httpPort)

		server := http.Server{Addr: fmt.Sprintf(":%v", httpPort), Handler: r}
		go server.ListenAndServe()

		log.Printf("HTTP SERVER STARTED IN GOROUTINE, waiting to die...")

		// block here until we get some sort of interrupt or kill
		<-done

		log.Printf("GOT SIGNAL TO DIE, cleaning up...")

		ctx := context.Background()
		ctxTimeout, cancel := context.WithTimeout(ctx, time.Second*5)
		defer cancel()

		err = server.Shutdown(ctxTimeout)
		if err != nil {
			log.Fatalf("Could not cleanly shut down http server: %v", err)
		}

		log.Printf("ALL DONE, GOODBYE")
	},
}
