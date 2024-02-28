package repeat

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"strconv"
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

func New(log *slog.Logger, requestGetter RequestGetter, proxyRT http.RoundTripper) echo.HandlerFunc {
	return func(c echo.Context) error {
		const op = "api.request.repeat.New"

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

		response, err := RepeatRequest([]byte(request.Request.Raw), proxyRT)

		if err != nil {
			log.Error("failed to RepeatRequest", sl.Err(err))

			c.JSON(http.StatusConflict, resp.Err(err.Error()))
			return err
		}
		contentType := "text/plain"
		if request.Response.Headers["Content-Type"] != nil {
			contentType = request.Response.Headers["Content-Type"][0]
		}
		log.Debug("TYPEE", "con", request.Response.Headers["Content-Type"])

		return c.Blob(http.StatusOK, contentType, response)
	}
}

func RepeatRequest(rawRequest []byte, proxyRT http.RoundTripper) ([]byte, error) {
	r, err := http.ReadRequest(bufio.NewReader(bytes.NewBuffer(rawRequest)))
	if err != nil {
		return nil, fmt.Errorf("error in client DO: %w", err)
	}
	r.RequestURI = ""
	if r.ContentLength == 0 {
		r.Body = nil
	}
	r.Header.Add("TrueProxy-Repeated", "TrueProxy")

	proxy.ChangeRequestToTarget(r, r.Host, proxy.HTTPS)

	client := http.Client{
		//Transport: http.DefaultTransport,
		Transport: proxyRT, // Save reapet to DB
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Timeout: 30 * time.Second,
	}

	resp, err := client.Do(r)
	if err != nil {
		return nil, fmt.Errorf("error in client DO: %w", err)
	}
	defer resp.Body.Close()

	if resp.Body != nil {
		b, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		err = resp.Body.Close()
		if err != nil {
			return nil, err
		}
		body := io.NopCloser(bytes.NewReader(b))
		resp.Body = body
		resp.ContentLength = int64(len(b))
		resp.Header.Set("Content-Length", strconv.Itoa(len(b)))
		return b, nil
	}

	return httputil.DumpResponse(resp, true)
}
