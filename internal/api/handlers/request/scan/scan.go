package scan

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	resp "github.com/mrdjeb/trueproxy/internal/api/response"
	"github.com/mrdjeb/trueproxy/internal/logger/sl"
	"github.com/mrdjeb/trueproxy/internal/models"
	"github.com/mrdjeb/trueproxy/internal/proxy"
)

type RequestGetter interface {
	ReadRequest(uint) (models.RequestResponse, error)
}

func New(log *slog.Logger, requestGetter RequestGetter) echo.HandlerFunc {
	return func(c echo.Context) error {
		const op = "api.request.scan.New"

		log = log.With(
			slog.String("op", op),
			slog.String("request_id", c.Request().Header.Get(echo.HeaderXRequestID)),
		)

		id, err := strconv.ParseUint(c.Param("id"), 10, 32)
		if err != nil {
			log.Error("failed to ParseUint ID", sl.Err(err))

			c.JSON(http.StatusBadRequest, resp.Err("bad id"))
			return err
		}

		request, err := requestGetter.ReadRequest(uint(id))
		if err != nil {
			log.Error("failed to requestListGetter", sl.Err(err))

			c.JSON(http.StatusInternalServerError, resp.Err("internal error"))
			return err
		}

		for _, promt := range dict {
			flag, err := CmdInjectionCheck([]byte(request.Request.Raw), promt)
			if err != nil {
				log.Error("failed to CmdInjectionCheck", sl.Err(err))

				c.JSON(http.StatusForbidden, resp.Err(err.Error()))
				return err
			}
			if flag {
				return c.JSON(http.StatusOK, "Command Injection Detected at "+promt)
			}
		}
		return c.JSON(http.StatusOK, "Not Detected")
	}
}

var dict = [...]string{
	";cat /etc/passwd;",
	"|cat /etc/passwd|",
	"`cat /etc/passwd`",
}

func CmdInjectionCheck(rawRequest []byte, bash string) (bool, error) {
	r, err := http.ReadRequest(bufio.NewReader(bytes.NewBuffer(rawRequest)))
	if err != nil {
		return false, fmt.Errorf("error in client DO: %w", err)
	}
	r.RequestURI = ""
	if r.ContentLength == 0 {
		r.Body = nil
	}
	proxy.ChangeRequestToTarget(r, r.Host, proxy.HTTPS)

	for k := range r.Header {
		r.Header[k] = append(r.Header[k], bash)
	}

	client := http.Client{
		Transport: http.DefaultTransport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Timeout: 30 * time.Second,
	}

	resp, err := client.Do(r)
	if err != nil {
		return false, fmt.Errorf("error in client DO: %w", err)
	}
	defer resp.Body.Close()

	if resp.Body != nil {
		b, err := io.ReadAll(resp.Body)
		if err != nil {
			return false, err
		}
		err = resp.Body.Close()
		if err != nil {
			return false, err
		}
		body := io.NopCloser(bytes.NewReader(b))
		resp.Body = body
		resp.ContentLength = int64(len(b))
		resp.Header.Set("Content-Length", strconv.Itoa(len(b)))

		return checkVulnerability(b), nil

	}
	return false, nil
}

func checkVulnerability(body []byte) bool {
	return strings.Contains(string(body), "root:")
}
