package proxy

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/mrdjeb/trueproxy/internal/logger/sl"
	"github.com/mrdjeb/trueproxy/internal/models"
	"github.com/mrdjeb/trueproxy/internal/storage"
)

type proxyRoundTripper struct {
	next http.RoundTripper
	log  *slog.Logger
	repo storage.RequestsRepo
}

func NewProxyRoundTripper(log *slog.Logger, repo storage.RequestsRepo) *proxyRoundTripper {
	return &proxyRoundTripper{
		next: http.DefaultTransport,
		log:  log,
		repo: repo,
	}
}

func (rt proxyRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	dumpRequest, err := httputil.DumpRequest(r, true)
	if err != nil {
		rt.log.Error("error while dump request %w", sl.Err(err))
	} else {
		rt.log.Info("Request dump", "request", fmt.Sprintf("[%s] %s %s\n", time.Now().Format(time.ANSIC), r.Method, r.URL.Host))
		//fmt.Sprintf("[%s] %s %s\n", time.Now().Format(time.ANSIC), r.Method, r.URL.String())
	}

	rDump := ParseRequest(r)
	if r.Body != nil {
		reader, err := r.GetBody()
		if err != nil {
			rt.log.Error("error GetBody", sl.Err(err))
		}
		bodyRaw, err := io.ReadAll(reader)
		if err != nil {
			rt.log.Error("error ReadAll", sl.Err(err))
		}
		rDump.Body = string(bodyRaw)
		reader.Close()
	}
	rDump.Raw = string(dumpRequest)

	resp, err := rt.next.RoundTrip(r)
	if err != nil {
		return resp, err
	}

	respDump := ParseResponse(resp)
	if resp.Body != nil {
		b, err := io.ReadAll(resp.Body)
		if err != nil {
			rt.log.Error("error ReadAll", sl.Err(err))
		}
		err = resp.Body.Close()
		if err != nil {
			rt.log.Error("error resp.Body.Clos", sl.Err(err))
		}
		body := io.NopCloser(bytes.NewReader(b))
		resp.Body = body
		resp.ContentLength = int64(len(b))
		resp.Header.Set("Content-Length", strconv.Itoa(len(b)))
		respDump.Body = string(b)
	}

	dumpResponse, err := httputil.DumpResponse(resp, true)
	if err != nil {
		rt.log.Error("error while dump response %w", sl.Err(err))
	} else {
		rt.log.Info("Response dump", "response", string(dumpResponse)) //fmt.Sprintf("[%s] %s %s %d\n", time.Now().Format(time.ANSIC), r.Method, r.URL.Host, resp.StatusCode))
		//fmt.Sprintf("[%s] %s %s %d\n", time.Now().Format(time.ANSIC), r.Method, r.URL.String(), resp.StatusCode)
	}
	respDump.Raw = string(dumpResponse)

	err = rt.repo.CreateRequest(
		&models.RequestResponse{
			Request:  *rDump,
			Response: *respDump,
		},
	)
	if err != nil {
		rt.log.Error("error while CreateRequest", sl.Err(err))
	}

	return resp, err

}

func ParseRequest(r *http.Request) *models.Request {
	reqD := &models.Request{
		Method:     r.Method,
		Path:       r.URL.Path,
		GetParams:  make(map[string][]string),
		Headers:    make(map[string][]string),
		Cookies:    make(map[string]string),
		PostParams: make(map[string][]string),
	}
	reqD.Host = r.Host //r.URL.Scheme + "://" + r.URL.Host

	getParamVals := make(url.Values)
	for k, values := range r.URL.Query() {
		getParamVals[k] = append(getParamVals[k], values...)
	}
	reqD.GetParams = getParamVals

	headers := make(http.Header)
	for k, values := range r.Header {
		headers[k] = append(headers[k], values...)
	}
	reqD.Headers = headers

	cookies := make(map[string]string)
	for _, v := range r.Cookies() {
		cookies[v.Name] = v.Value
	}
	reqD.Cookies = cookies

	if err := r.ParseForm(); err == nil {
		postFormVals := make(url.Values)
		for k, values := range r.PostForm {
			postFormVals[k] = append(postFormVals[k], values...)
		}
		reqD.PostParams = postFormVals
	}
	return reqD
}

func ParseResponse(r *http.Response) *models.Response {
	reqD := &models.Response{
		StatusCode: r.StatusCode,
		Headers:    make(map[string][]string),
		Cookies:    make(map[string]string),
		PostParams: make(map[string][]string),
	}

	headers := make(http.Header)
	for k, values := range r.Header {
		headers[k] = append(headers[k], values...)
	}
	reqD.Headers = headers

	cookies := make(map[string]string)
	for _, v := range r.Cookies() {
		cookies[v.Name] = v.Value
	}
	reqD.Cookies = cookies

	return reqD
}

func Decode(ri *models.Request) (*http.Request, error) {
	var body io.Reader
	if ri.Body != "" {
		body = strings.NewReader(ri.Body)
	}

	r, err := http.NewRequest(
		ri.Method,
		fmt.Sprintf("http://%s%s", ri.Host, ri.Path),
		body,
	)
	if err != nil {
		return nil, err
	}

	query := r.URL.Query()
	for param, valList := range ri.GetParams {
		for _, val := range valList {
			query.Add(param, val)
		}
	}
	r.URL.RawQuery = query.Encode()

	for name, val := range ri.Cookies {
		r.AddCookie(&http.Cookie{Name: name, Value: val})
	}

	for header, headerValList := range ri.Headers {
		for _, headerVal := range headerValList {
			r.Header.Add(header, headerVal)
		}
	}

	for param, valList := range ri.PostParams {
		for _, val := range valList {
			r.PostForm.Add(param, val)
		}
	}

	return r, nil
}
