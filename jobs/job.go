package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gocraft/work"
	"github.com/gomodule/redigo/redis"
	"github.com/joho/godotenv"
	"github.com/resend/resend-go/v3"
)

var redisPool = &redis.Pool{
	MaxActive: 5,
	MaxIdle:   5,
	Wait:      true,
	Dial: func() (redis.Conn, error) {
		return redis.Dial("tcp", ":6379")
	},
}

var emailQueue = work.NewEnqueuer("email", redisPool)

type Context struct {
	email  string
	userId int64
}

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Println("Warning: unable to find .env file")
	}

	_, err = emailQueue.Enqueue("send_email", work.Q{"email_address": "esansamuel555@gmail.com", "user_id": 123})
	if err != nil {
		log.Fatal(err)
	}

	pool := work.NewWorkerPool(Context{}, 10, "email", redisPool)

	pool.Middleware((*Context).Log)
	pool.Middleware((*Context).FindUser)

	pool.Job("send_email", (*Context).sendEmailToUser)
	pool.Start()

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)
	<-signalChan

	pool.Stop()
}

func (c *Context) Log(job *work.Job, next work.NextMiddlewareFunc) error {
	fmt.Printf("Background Job is processing. Job Id:", job.ID, job.Name)
	return next()
}

func (c *Context) FindUser(job *work.Job, next work.NextMiddlewareFunc) error {
	if _, ok := job.Args["user_id"]; ok {
		c.email = job.ArgString("email_address")
		c.userId = job.ArgInt64("user_id")
		if err := job.ArgError(); err != nil {
			return err
		}
	}
	return next()
}

func (c *Context) sendEmailToUser(job *work.Job) error {
	RESEND_API_KEY := os.Getenv("RESEND_API_KEY")

	client := resend.NewClient(RESEND_API_KEY)
	fmt.Println("Email:", c.email, job.ArgString("email_address"))
	emailAddr := c.email

	params := &resend.SendEmailRequest{
		From:    "Acme <onboarding@resend.dev>",
		To:      []string{emailAddr},
		Html:    "<strong>hello world</strong>",
		Subject: "Hello from Golang",
		Cc:      []string{"cc@example.com"},
		Bcc:     []string{"bcc@example.com"},
		ReplyTo: "replyto@example.com",
	}

	sent, err := client.Emails.Send(params)
	if err != nil {
		fmt.Println(err.Error())
		return err
	}
	fmt.Println(sent.Id)
	return nil
}
