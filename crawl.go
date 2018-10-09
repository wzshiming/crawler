package crawler

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"reflect"
	"sync"
	"time"

	"github.com/wzshiming/inject"
	"github.com/wzshiming/requests"
	"github.com/wzshiming/task"
	ffmt "gopkg.in/ffmt.v1"
)

type Crawler struct {
	stepSet  map[string]interface{}
	interval time.Duration
	tasks    *task.Task
	inj      *inject.Injector
	cli      *requests.Client
	log      *log.Logger
	nextTime time.Time
	mux      sync.Mutex
}

func NewCrawler() *Crawler {
	c := &Crawler{
		stepSet: map[string]interface{}{},
		tasks:   task.NewTask(2),
		inj:     inject.NewInjector(nil),
		cli:     requests.NewClient().SetCache(requests.FileCacheDir("tmp/")).SetTimeout(time.Second * 5),
		log:     log.New(os.Stdout, "CRAWLER", log.Ldate),
	}
	c.Map(&c)
	c.Map(&c.inj)
	c.Map(&c.cli)
	c.Map(&c.log)
	c.Map(&c.tasks)
	c.inj = c.inj.Child()
	return c
}

func (c *Crawler) SetInterval(interval time.Duration) {
	c.interval = interval
}

func (c *Crawler) Map(v interface{}) error {
	c.mux.Lock()
	defer c.mux.Unlock()
	return c.inj.Map(reflect.ValueOf(v))
}

func (c *Crawler) Step(name string, fun interface{}) {
	c.stepSet[name] = fun
}

func (c *Crawler) NextStep(fun interface{}, args ...interface{}) error {
	if name, ok := fun.(string); ok {
		fun, ok = c.stepSet[name]
		if !ok {
			return fmt.Errorf("Error: No steps exist: %s", name)
		}
	}
	inj := c.inj.Child()
	for _, v := range args {
		err := inj.Map(reflect.ValueOf(v))
		if err != nil {
			return err
		}
	}

	c.nextTime = lastTime(c.nextTime.Add(c.interval), time.Now())
	c.tasks.Add(c.nextTime, func() {
		c.mux.Lock()
		defer c.mux.Unlock()
		_, err := inj.Call(reflect.ValueOf(fun))
		if err != nil {
			ffmt.Mark(err)
			return
		}
	})
	return nil
}

func (c *Crawler) Log(a ...interface{}) {
	c.log.Println(a...)
}

func (c *Crawler) CookieJar() http.CookieJar {
	return c.cli.GetCookieJar()
}

func (c *Crawler) Request() *requests.Request {
	return c.cli.NewRequest()
}

func (c *Crawler) Wait() {
	c.tasks.Join()
}

func lastTime(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}
