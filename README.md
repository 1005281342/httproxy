# httproxy
http服务反向代理

## 环境
### 北极星
[单机版安装](http://polarismesh.cn/zh/doc/%E5%BF%AB%E9%80%9F%E5%85%A5%E9%97%A8/%E5%AE%89%E8%A3%85%E6%9C%8D%E5%8A%A1%E7%AB%AF/%E5%AE%89%E8%A3%85%E5%8D%95%E6%9C%BA%E7%89%88.html#%E5%8D%95%E6%9C%BA%E7%89%88%E5%AE%89%E8%A3%85)

### go
1.16+

## 启动

### 启动grpchttp/example/hello
```shell script
cd grpchttp/example/hello
go run hello.go -f ./etc/hello.yaml
```

### 启动proxy
`go run proxy.go`

## 测试
```shell script
curl --location --request POST '127.0.0.1:2333/hello.Hello/Say' \
--header 'Content-Type: application/json' \
--data-raw '{
    "name": "test"
}'
```

## 使用grpchttp给grpc服务注册HTTP接口 以grpchttp/example/hello为例
1. 在stub代码目录下添加文件grpchttp.go(其他合法名字也可以)。
2. 在grpchttp.go文件中添加可导出对象ServiceDesc代码：
    ```shell script
    package hello
    
    var ServiceDesc = _Hello_serviceDesc
    ```
3. 在hello.go中注册HTTP接口
    ```shell script
    var mux http.ServeMux
    httpgrpc.HandleServices(mux.HandleFunc, "/", reg, nil, nil)
    lis, err := net.Listen("tcp", fmt.Sprintf(":%d", c.HttpPort))
    if err != nil {
        panic(err)
    }
    logx.Infof("http port: %s", lis.Addr().String())

    httpServer := http.Server{Handler: &mux}
    go httpServer.Serve(lis)
    defer httpServer.Close()
    ```
