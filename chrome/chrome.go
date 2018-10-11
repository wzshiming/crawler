package chrome

import (
	"context"
	"log"
	"time"
	"unsafe"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"github.com/chromedp/chromedp/runner"
	"github.com/wzshiming/ffmt"
)

type Chrome struct {
	cdp  *chromedp.CDP
	ctx  context.Context
	exit func() error
}

func NewChrome() (*Chrome, error) {
	ctx, cancel := context.WithCancel(context.Background())

	cdp, err := chromedp.New(ctx,
		chromedp.WithLog(log.Printf),
		// chromedp.WithTargets(
		// 	client.New(
		// 		client.WatchTimeout(time.Second*15),
		// 		client.WatchInterval(time.Second),
		// 	).
		// 		WatchPageTargets(ctx),
		// ),
		chromedp.WithRunnerOptions(
			runner.WindowSize(1920, 1080),
			runner.Headless,
			runner.DisableGPU,
			runner.NoSandbox,
			runner.NoFirstRun,
		),
	)

	//	chromedp.WithLog(log.Printf),
	if err != nil {
		ffmt.Mark(err)
		return nil, err
	}

	exit := func() error {
		// shutdown chrome
		err := cdp.Shutdown(ctx)
		if err != nil {
			return err
		}
		// wait for chrome to finish
		err = cdp.Wait()
		if err != nil {
			return err
		}
		cancel()
		return nil
	}

	return &Chrome{
		cdp:  cdp,
		ctx:  ctx,
		exit: exit,
	}, nil
}

func (c *Chrome) Shutdown() error {
	return c.exit()
}

func (c *Chrome) HTML(url string) ([]byte, error) {
	var res string
	err := c.cdp.Run(c.ctx, chromedp.Tasks{
		chromedp.Navigate(url),
		chromedp.OuterHTML("html", &res, chromedp.ByQuery),
	})
	if err != nil {
		ffmt.Mark(err)
		return nil, err
	}
	return *(*[]byte)(unsafe.Pointer(&res)), nil
}

func (c *Chrome) PDF(url string) ([]byte, error) {
	var res []byte

	err := c.cdp.Run(c.ctx, chromedp.Tasks{
		chromedp.Navigate(url),
		chromedp.Sleep(time.Second * 2),
		chromedp.ActionFunc(func(ctxt context.Context, h cdp.Executor) error {
			buf, err := page.PrintToPDF().
				WithMarginTop(0.01).
				WithMarginBottom(0.01).
				WithMarginRight(0.01).
				WithMarginLeft(0.01).
				WithPreferCSSPageSize(true).
				WithPrintBackground(true).
				WithLandscape(true).
				Do(ctxt, h)
			if err != nil {
				return err
			}
			res = buf
			return nil
		}),
	})
	if err != nil {
		ffmt.Mark(err)
		return nil, err
	}
	return res, nil
}

func (c *Chrome) Screenshot(url string) ([]byte, error) {
	var res []byte
	err := c.cdp.Run(c.ctx, chromedp.Tasks{
		chromedp.Navigate(url),
		chromedp.Sleep(time.Second * 2),
		chromedp.ActionFunc(func(ctx context.Context, h cdp.Executor) error {
			scr, err := page.CaptureScreenshot().
				Do(ctx, h)
			if err != nil {
				return err
			}
			res = scr
			return nil
		}),
	})

	if err != nil {
		ffmt.Mark(err)
		return nil, err
	}
	return res, nil
}
