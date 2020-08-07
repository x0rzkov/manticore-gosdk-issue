# manticore-issue with length of strings

### Pre-requisites
- docker
- docker-compose

### Installation

```bash
cd ./data
tar xvf manticore_papers.sql.tar.gz
```

### Run

```bash
$ docker-compose up -d manticore
$ docker-compose exec manticore sh -c 'mysql -P9306'
$ mysql> source /opt/data/manticore_papers.sql
$ mysql> exit
$ docker-compose up -d
```

#### First bug: slice bounds out of range...
```
$ open http://[YOUR_PUBLIC_IP]:8086/search/l
```

#### Second bug: Sphinxql: failed to read searchd response
```
$ go run main.go
$ open http://[YOUR_PUBLIC_IP]:8086/rss/?f_frameworks=1
$ # it works
```

```
docker-compose up 
$ open http://[YOUR_PUBLIC_IP]:8086/rss/?f_frameworks=1
$ # does not work
```


## description
```bash
runtime error: slice bounds out of range [:774778400] with capacity 22412
/usr/local/go/src/runtime/panic.go:106 (0x432eb2)
	goPanicSliceAcap: panic(boundsError{x: int64(x), signed: true, y: y, code: boundsSliceAcap})
/home/ubuntu/lucmichalski/paper2code/pkg/manticore/sphinxql.go:311 (0x10a139c)
	(*apibuf).getMysqlStrLen: result := string((*buf)[:lng])
/home/ubuntu/lucmichalski/paper2code/pkg/manticore/sphinxql.go:234 (0x10a0a5a)
	(*Sqlresult).parserow: strValue := buf.getMysqlStrLen()
/home/ubuntu/lucmichalski/paper2code/pkg/manticore/sphinxql.go:74 (0x109f901)
	(*Sqlresult).parseChain: if !rs.parserow(source) {
/home/ubuntu/lucmichalski/paper2code/pkg/manticore/sphinxql.go:20 (0x10a62b5)
	parseSphinxqlAnswer.func1: if rs.parseChain(answer) {
/home/ubuntu/lucmichalski/paper2code/pkg/manticore/client.go:225 (0x1093649)
	(*Client).netQuery: return parser(&answer), nil
/home/ubuntu/lucmichalski/paper2code/pkg/manticore/manticore.go:398 (0x109503f)
	(*Client).Sphinxql: blob, err := cl.netQuery(commandSphinxql,
/home/ubuntu/lucmichalski/paper2code/main.go:3639 (0x118b2a7)
	controllersSearch: res2, err2 := cl.Sphinxql(query)
/home/ubuntu/go/pkg/mod/github.com/gin-gonic/gin@v1.6.3/context.go:161 (0xc9b61a)
	(*Context).Next: c.handlers[c.index](c)
/home/ubuntu/go/pkg/mod/github.com/gin-gonic/gin@v1.6.3/recovery.go:83 (0xcaedaf)
	RecoveryWithWriter.func1: c.Next()
/home/ubuntu/go/pkg/mod/github.com/gin-gonic/gin@v1.6.3/context.go:161 (0xc9b61a)
	(*Context).Next: c.handlers[c.index](c)
/home/ubuntu/go/pkg/mod/github.com/gin-gonic/gin@v1.6.3/logger.go:241 (0xcadee0)
	LoggerWithConfig.func1: c.Next()
/home/ubuntu/go/pkg/mod/github.com/gin-gonic/gin@v1.6.3/context.go:161 (0xc9b61a)
	(*Context).Next: c.handlers[c.index](c)
/home/ubuntu/go/pkg/mod/github.com/gin-gonic/gin@v1.6.3/gin.go:409 (0xca53f5)
	(*Engine).handleHTTPRequest: c.Next()
/home/ubuntu/go/pkg/mod/github.com/gin-gonic/gin@v1.6.3/gin.go:367 (0xca4b0c)
	(*Engine).ServeHTTP: engine.handleHTTPRequest(c)
/home/ubuntu/go/pkg/mod/github.com/qor/session@v0.0.0-20170907035918-8206b0adab70/gorilla/gorilla.go:125 (0xb2b591)
	Gorilla.Middleware.func1: handler.ServeHTTP(w, req.WithContext(ctx))
/usr/local/go/src/net/http/server.go:2012 (0x76eb73)
	HandlerFunc.ServeHTTP: f(w, r)
/usr/local/go/src/net/http/server.go:2807 (0x772002)
	serverHandler.ServeHTTP: handler.ServeHTTP(rw, req)
/usr/local/go/src/net/http/server.go:1895 (0x76d97b)
	(*conn).serve: serverHandler{c.server}.ServeHTTP(w, w.req)
/usr/local/go/src/runtime/asm_amd64.s:1373 (0x465670)
	goexit: BYTE	$0x90	// NOP
```
