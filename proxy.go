package httproxy

import (
	"bytes"
	"context"
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
	rkentry "github.com/rookie-ninja/rk-entry/v2/entry"
	rkprom "github.com/rookie-ninja/rk-prom"
)

const (
	errMsgKey = "Err_msg"
	ErrMsgKey = errMsgKey

	snSuffix = "-http"
	SnSuffix = snSuffix
)

// Res 返回
type Res struct {
	Code   int         `json:"code"`
	Result bool        `json:"result"`
	Info   interface{} `json:"info"`
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

type OptionConf struct {
	promBootYamlPath string
	metricsSet       *rkprom.MetricsSet
	namespace        string
}

type Option func(cfg *OptionConf)

func (o *OptionConf) Report(pathKey string) {
	if o.metricsSet == nil {
		return
	}

	var c = o.metricsSet.GetCounterWithValues(pathKey)
	if c != nil {
		c.Inc()
		return
	}

	if err := o.metricsSet.RegisterCounter(pathKey); err != nil {
		log.Printf("注册%s的Counter失败:%+v", pathKey, err)
		return
	}
	o.metricsSet.GetCounterWithValues(pathKey).Inc()
}

func WithProm(namespace string, path string) Option {

	return func(cfg *OptionConf) {
		cfg.promBootYamlPath = path
		if namespace == "" {
			cfg.namespace = path
		}

		if cfg.metricsSet == nil {
			go func() {
				maps := rkprom.RegisterPromEntriesWithConfig(path)

				entry := maps[rkprom.PromEntryNameDefault].(*rkprom.PromEntry)
				cfg.metricsSet = rkprom.NewMetricsSet(cfg.namespace, "httproxy", entry.Registerer)

				entry.Bootstrap(context.Background())

				rkentry.GlobalAppCtx.WaitForShutdownSig()

				// stop server
				entry.Interrupt(context.Background())
			}()
		}
	}
}

func pathKey(path string, failed bool) string {
	const (
		suffixFailed = "_failed"
	)
	var pk = strings.Replace(strings.Trim(path, "/"), "/", "_", -1)
	pk = strings.Replace(pk, ".", "", -1)
	pk = strings.Replace(pk, "-", "", -1)
	if failed {
		pk += suffixFailed
	}
	return pk
}

func New(namespace string, gConsumer api.ConsumerAPI, opts ...Option) *httputil.ReverseProxy {
	var cfg = &OptionConf{}
	for _, opt := range opts {
		opt(cfg)
	}

	director := func(req *http.Request) {

		var errMsg string
		defer func() {
			// 总请求次数++
			cfg.Report(pathKey(req.URL.Path, false))

			if errMsg != "" {
				var target, _ = url.Parse(fmt.Sprintf("http://127.0.0.1:%s/err", errPort))
				req.URL = target

				var eb = errBody{Err: errMsg}
				var e, _ = json.Marshal(eb)

				req.Header.Add(errMsgKey, string(e))

				// 转发到了errHandler说明一次请求处理失败了
				// TODO 细分错误分别上报
				// 用户输入参数不符合规范或系统服务故障的错误次数++
				cfg.Report(pathKey(req.URL.Path, false))
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
		// 业务服务处理异常次数++
		cfg.Report(pathKey(req.URL.Path, true))
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
