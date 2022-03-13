package httproxy

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/polarismesh/polaris-go/api"
)

const (
	errMsgKey = "Err_msg"
	ErrMsgKey = errMsgKey

	snSuffix = "-http"
	SnSuffix = snSuffix
)

// Res 返回
type Res struct {
	Code   int
	Result bool
	Info   interface{}
}

var errPort string

func init() {
	go func() {
		http.HandleFunc("/err", errHandle)

		lis, err := net.Listen("tcp", "0.0.0.0:0")
		if err != nil {
			panic(err)
		}
		var a = strings.Split(lis.Addr().String(), ":")
		errPort = a[len(a)-1]

		http.Serve(lis, nil)
	}()
}

func errHandle(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, r.Header.Get(errMsgKey)) //这个写入到w的是输出到客户端的
	r.Header.Del(errMsgKey)
}

type errBody struct {
	Err string `json:"err"`
}

func New(namespace string, gConsumer api.ConsumerAPI) *httputil.ReverseProxy {

	director := func(req *http.Request) {

		var errMsg string
		defer func() {
			if errMsg != "" {
				var target, _ = url.Parse(fmt.Sprintf("http://127.0.0.1:%s/err", errPort))
				req.URL = target

				var eb = errBody{Err: errMsg}
				var e, _ = json.Marshal(eb)

				req.Header.Add(errMsgKey, string(e))
			}
		}()

		if req.Method == http.MethodOptions {
			errMsg = "options请求处理"
			return
		}

		if req.Method != http.MethodPost {
			errMsg = fmt.Sprintf("不支持%s请求", req.Method)
			return
		}

		if req.URL == nil {
			errMsg = "url is nil"
			return
		}

		const sepSymbol = "/"

		// 解析参数
		log.Printf("path: %s", req.URL.Path)
		req.URL.Path = strings.TrimRight(req.URL.Path, sepSymbol)
		var infos = strings.Split(strings.TrimLeft(req.URL.Path, sepSymbol), sepSymbol)
		if len(infos) != 2 {
			errMsg = `请求格式不规范 示例 "method": "POST", "url": "127.0.0.1:2333/serviceName/methodName"`
			return
		}
		var serviceName = infos[0]
		if len(serviceName) < len(snSuffix) {
			serviceName += snSuffix
		} else if serviceName[len(serviceName)-len(snSuffix):] != snSuffix {
			serviceName += snSuffix
		}

		infos[0] = serviceName[:len(serviceName)-len(snSuffix)]
		req.URL.Path = strings.Join(infos, sepSymbol)

		getOneRequest := &api.GetOneInstanceRequest{}
		getOneRequest.Namespace = namespace
		getOneRequest.Service = serviceName
		oneInstResp, err := gConsumer.GetOneInstance(getOneRequest)
		if err != nil {
			errMsg = fmt.Sprintf("fail to getOneInstance, err is %v", err)
			return
		}
		instance := oneInstResp.GetInstance()
		if instance == nil {
			errMsg = "no instance"
			return
		}
		log.Printf("instance getOneInstance is %s:%d \n", instance.GetHost(), instance.GetPort())

		if instance.GetProtocol() != "http" {
			errMsg = "instance Protocol not is http"
			return
		}

		var lo = fmt.Sprintf("http://%s:%d/", instance.GetHost(), instance.GetPort())
		var target, tErr = url.Parse(lo)
		if tErr != nil {
			errMsg = tErr.Error()
			return
		}
		var targetQuery = target.RawQuery
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.URL.Path, req.URL.RawPath = joinURLPath(target, req.URL)
		if targetQuery == "" || req.URL.RawQuery == "" {
			req.URL.RawQuery = targetQuery + req.URL.RawQuery
		} else {
			req.URL.RawQuery = targetQuery + "&" + req.URL.RawQuery
		}

		if _, ok := req.Header["User-Agent"]; !ok {
			req.Header.Set("User-Agent", "")
		}
	}

	modifyFunc := func(res *http.Response) error {
		if res.StatusCode != http.StatusOK {
			return errors.New(fmt.Sprintf("业务方错误，信息：%s", res.Status))
		}

		var (
			err        error
			oldPayload []byte
		)
		if oldPayload, err = ioutil.ReadAll(res.Body); err != nil {
			return err
		}

		var info = make(map[string]interface{})
		if err = json.Unmarshal(oldPayload, &info); err != nil {
			return err
		}

		var (
			newPayLoadRes = Res{
				Result: true,
				Code:   http.StatusOK,
				Info:   info,
			}
			newPayLoad []byte
		)

		if newPayLoad, err = json.Marshal(newPayLoadRes); err != nil {
			return err
		}
		res.Body = ioutil.NopCloser(bytes.NewBuffer(newPayLoad))
		res.ContentLength = int64(len(newPayLoad))
		res.Header.Set("Content-Length", fmt.Sprint(len(newPayLoad)))

		return nil
	}

	errorHandler := func(res http.ResponseWriter, req *http.Request, err error) {
		res.Write([]byte(err.Error()))
	}

	return &httputil.ReverseProxy{Director: director, ModifyResponse: modifyFunc, ErrorHandler: errorHandler}
}

func joinURLPath(a, b *url.URL) (string, string) {
	if a.RawPath == "" && b.RawPath == "" {
		return singleJoiningSlash(a.Path, b.Path), ""
	}
	// Same as singleJoiningSlash, but uses EscapedPath to determine
	// whether a slash should be added
	aPath := a.EscapedPath()
	bPath := b.EscapedPath()

	aSlash := strings.HasSuffix(aPath, "/")
	bSlash := strings.HasPrefix(bPath, "/")

	switch {
	case aSlash && bSlash:
		return a.Path + b.Path[1:], aPath + bPath[1:]
	case !aSlash && !bSlash:
		return a.Path + "/" + b.Path, aPath + "/" + bPath
	}
	return a.Path + b.Path, aPath + bPath
}

func singleJoiningSlash(a, b string) string {
	aSlash := strings.HasSuffix(a, "/")
	bSlash := strings.HasPrefix(b, "/")
	switch {
	case aSlash && bSlash:
		return a + b[1:]
	case !aSlash && !bSlash:
		return a + "/" + b
	}
	return a + b
}
