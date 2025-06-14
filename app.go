package main

import (
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/mycoool/gohook/internal/hook"
	"github.com/mycoool/gohook/internal/pidfile"
	"github.com/mycoool/gohook/internal/ver"

	"github.com/fsnotify/fsnotify"
	"github.com/gin-gonic/gin"
	"github.com/mycoool/gohook/internal/router"
	"github.com/mycoool/gohook/internal/stream"
)

var (
	ip                 = flag.String("ip", "0.0.0.0", "ip the webhook should serve hooks on")
	port               = flag.Int("port", 9000, "port the webhook should serve hooks on")
	verbose            = flag.Bool("verbose", false, "show verbose output")
	logPath            = flag.String("logfile", "", "send log output to a file; implicitly enables verbose logging")
	debug              = flag.Bool("debug", false, "show debug output")
	ginDebug           = flag.Bool("gin-debug", false, "show gin debug output")
	noPanic            = flag.Bool("nopanic", false, "do not panic if hooks cannot be loaded when webhook is not running in verbose mode")
	hotReload          = flag.Bool("hotreload", false, "watch hooks file for changes and reload them automatically")
	hooksURLPrefix     = flag.String("urlprefix", "hooks", "url prefix to use for served hooks (protocol://yourserver:port/PREFIX/:hook-id)")
	secure             = flag.Bool("secure", false, "use HTTPS instead of HTTP")
	asTemplate         = flag.Bool("template", false, "parse hooks file as a Go template")
	cert               = flag.String("cert", "cert.pem", "path to the HTTPS certificate pem file")
	key                = flag.String("key", "key.pem", "path to the HTTPS certificate private key pem file")
	justDisplayVersion = flag.Bool("version", false, "display webhook version and quit")
	justListCiphers    = flag.Bool("list-cipher-suites", false, "list available TLS cipher suites")
	tlsMinVersion      = flag.String("tls-min-version", "1.2", "minimum TLS version (1.0, 1.1, 1.2, 1.3)")
	tlsCipherSuites    = flag.String("cipher-suites", "", "comma-separated list of supported TLS cipher suites")
	maxMultipartMem    = flag.Int64("max-multipart-mem", 1<<20, "maximum memory in bytes for parsing multipart form data before disk caching")
	httpMethods        = flag.String("http-methods", "", `set default allowed HTTP methods (ie. "POST"); separate methods with comma`)
	pidPath            = flag.String("pidfile", "", "create PID file at the given path")

	responseHeaders hook.ResponseHeaders
	hooksFiles      hook.HooksFiles

	loadedHooksFromFiles = make(map[string]hook.Hooks)

	watcher *fsnotify.Watcher
	signals chan os.Signal
	pidFile *pidfile.PIDFile
	setUID  = 0
	setGID  = 0
	socket  = ""
	addr    = ""
)

func matchLoadedHook(id string) *hook.Hook {
	if router.HookManager != nil {
		return router.HookManager.MatchLoadedHook(id)
	}

	// 回退到原有逻辑
	for _, hooks := range loadedHooksFromFiles {
		if hook := hooks.Match(id); hook != nil {
			return hook
		}
	}

	return nil
}

func lenLoadedHooks() int {
	if router.HookManager != nil {
		return router.HookManager.LenLoadedHooks()
	}

	// 回退到原有逻辑
	sum := 0
	for _, hooks := range loadedHooksFromFiles {
		sum += len(hooks)
	}

	return sum
}

func main() {
	flag.Var(&hooksFiles, "hooks", "path to the json file containing defined hooks the webhook should serve, use multiple times to load from different files")
	flag.Var(&responseHeaders, "header", "response header to return, specified in format name=value, use multiple times to set multiple headers")

	// register platform-specific flags
	platformFlags()

	flag.Parse()

	if *justDisplayVersion {
		fmt.Println("webhook version " + ver.Version)
		os.Exit(0)
	}

	if *justListCiphers {
		err := writeTLSSupportedCipherStrings(os.Stdout, getTLSMinVersion(*tlsMinVersion))
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	if (setUID != 0 || setGID != 0) && (setUID == 0 || setGID == 0) {
		fmt.Println("error: setuid and setgid options must be used together")
		os.Exit(1)
	}

	if *debug || *logPath != "" {
		*verbose = true
	}

	if len(hooksFiles) == 0 {
		hooksFiles = append(hooksFiles, "hooks.json")
	}

	// logQueue is a queue for log messages encountered during startup. We need
	// to queue the messages so that we can handle any privilege dropping and
	// log file opening prior to writing our first log message.
	var logQueue []string

	// by default the listen address is ip:port (default 0.0.0.0:9000), but
	// this may be modified by trySocketListener
	addr = fmt.Sprintf("%s:%d", *ip, *port)

	ln, err := trySocketListener()
	if err != nil {
		logQueue = append(logQueue, fmt.Sprintf("error listening on socket: %s", err))
		// we'll bail out below
	} else if ln == nil {
		// Open listener early so we can drop privileges.
		ln, err = net.Listen("tcp", addr)
		if err != nil {
			logQueue = append(logQueue, fmt.Sprintf("error listening on port: %s", err))
			// we'll bail out below
		}
	}

	if setUID != 0 {
		err := dropPrivileges(setUID, setGID)
		if err != nil {
			logQueue = append(logQueue, fmt.Sprintf("error dropping privileges: %s", err))
			// we'll bail out below
		}
	}

	if *logPath != "" {
		file, err := os.OpenFile(*logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o666)
		if err != nil {
			logQueue = append(logQueue, fmt.Sprintf("error opening log file %q: %v", *logPath, err))
			// we'll bail out below
		} else {
			log.SetOutput(file)
		}
	}

	log.SetPrefix("[webhook] ")
	log.SetFlags(log.Ldate | log.Ltime)

	if len(logQueue) != 0 {
		for i := range logQueue {
			log.Println(logQueue[i])
		}

		os.Exit(1)
	}

	if !*verbose {
		log.SetOutput(io.Discard)
	}

	// Create pidfile
	if *pidPath != "" {
		var err error

		pidFile, err = pidfile.New(*pidPath)
		if err != nil {
			log.Fatalf("Error creating pidfile: %v", err)
		}

		defer func() {
			// NOTE(moorereason): my testing shows that this doesn't work with
			// ^C, so we also do a Remove in the signal handler elsewhere.
			if nerr := pidFile.Remove(); nerr != nil {
				log.Print(nerr)
			}
		}()
	}

	log.Println("version " + ver.Version + " starting")

	// set os signal watcher
	setupSignals()

	// load and parse hooks
	for _, hooksFilePath := range hooksFiles {
		log.Printf("attempting to load hooks from %s\n", hooksFilePath)

		newHooks := hook.Hooks{}

		err := newHooks.LoadFromFile(hooksFilePath, *asTemplate)

		if err != nil {
			log.Printf("couldn't load hooks from file! %+v\n", err)
		} else {
			log.Printf("found %d hook(s) in file\n", len(newHooks))

			for _, hook := range newHooks {
				if matchLoadedHook(hook.ID) != nil {
					log.Fatalf("error: hook with the id %s has already been loaded!\nplease check your hooks file for duplicate hooks ids!\n", hook.ID)
				}
				log.Printf("\tloaded: %s\n", hook.ID)
			}

			loadedHooksFromFiles[hooksFilePath] = newHooks
		}
	}

	newHooksFiles := hooksFiles[:0]
	for _, filePath := range hooksFiles {
		if _, ok := loadedHooksFromFiles[filePath]; ok {
			newHooksFiles = append(newHooksFiles, filePath)
		}
	}

	hooksFiles = newHooksFiles

	if !*verbose && !*noPanic && lenLoadedHooks() == 0 {
		log.SetOutput(os.Stdout)
		log.Fatalln("couldn't load any hooks from file!\naborting webhook execution since the -verbose flag is set to false.\nIf, for some reason, you want webhook to start without the hooks, either use -verbose flag, or -nopanic")
	}

	if *hotReload {
		var err error

		watcher, err = fsnotify.NewWatcher()
		if err != nil {
			log.Fatal("error creating file watcher instance\n", err)
		}
		defer watcher.Close()

		for _, hooksFilePath := range hooksFiles {
			// set up file watcher
			log.Printf("setting up file watcher for %s\n", hooksFilePath)

			err = watcher.Add(hooksFilePath)
			if err != nil {
				log.Print("error adding hooks file to the watcher\n", err)
				return
			}
		}

		go watchForFileChange()
	}

	// 根据debug参数设置gin模式
	if *ginDebug {
		gin.SetMode(gin.DebugMode)
		log.Printf("running in debug mode")
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// 设置router对hooks数据的引用
	router.LoadedHooksFromFiles = &loadedHooksFromFiles

	// 初始化HookManager
	router.HookManager = hook.NewHookManager(&loadedHooksFromFiles, hooksFiles, *asTemplate)

	r := router.InitRouter()

	// 启用方法不允许处理
	r.HandleMethodNotAllowed = true

	// 设置gin中间件
	if *debug {
		// debug模式下使用详细的日志中间件
		r.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
			return fmt.Sprintf("[GIN] %v | %3d | %13v | %15s | %-7s %#v\n",
				param.TimeStamp.Format("2006/01/02 - 15:04:05"),
				param.StatusCode,
				param.Latency,
				param.ClientIP,
				param.Method,
				param.Path,
			)
		}))
	} else {
		// 生产模式下使用简洁的日志
		r.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
			return fmt.Sprintf("[GIN] %3d | %13v | %s | %s %s\n",
				param.StatusCode,
				param.Latency,
				param.ClientIP,
				param.Method,
				param.Path,
			)
		}))
	}

	r.Use(gin.Recovery())

	// Clean up input
	*httpMethods = strings.ToUpper(strings.ReplaceAll(*httpMethods, " ", ""))

	// 注意：根路径 "/" 现在由前端UI路由处理（在router.InitRouter()中注册）

	// webhook路由 - 支持所有HTTP方法
	hooksPath := "/" + *hooksURLPrefix + "/*id"
	r.Any(hooksPath, ginHookHandler)

	// Create common HTTP server settings
	svr := &http.Server{
		Handler: r,
	}

	// Serve HTTP
	if !*secure {
		log.Printf("serving hooks on http://%s%s", addr, makeHumanPattern(hooksURLPrefix))
		log.Print(svr.Serve(ln))

		return
	}

	// Server HTTPS
	svr.TLSConfig = &tls.Config{
		CipherSuites:             getTLSCipherSuites(*tlsCipherSuites),
		CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
		MinVersion:               getTLSMinVersion(*tlsMinVersion),
		PreferServerCipherSuites: true,
	}
	svr.TLSNextProto = make(map[string]func(*http.Server, *tls.Conn, http.Handler)) // disable http/2

	log.Printf("serving hooks on https://%s%s", addr, makeHumanPattern(hooksURLPrefix))
	log.Print(svr.ServeTLS(ln, *cert, *key))
}

func ginHookHandler(c *gin.Context) {
	req := &hook.Request{
		ID:         c.GetString("request-id"), // 可以通过中间件设置
		RawRequest: c.Request,
	}

	// 如果没有request-id，生成一个简单的ID
	if req.ID == "" {
		req.ID = fmt.Sprintf("%d", time.Now().UnixNano())
	}

	log.Printf("[%s] incoming HTTP %s request from %s\n", req.ID, c.Request.Method, c.Request.RemoteAddr)

	// debug模式下输出更多请求信息
	if *debug {
		log.Printf("[%s] Request Headers: %v", req.ID, c.Request.Header)
		log.Printf("[%s] Request URL: %s", req.ID, c.Request.URL.String())
		log.Printf("[%s] User-Agent: %s", req.ID, c.Request.UserAgent())
	}

	// 获取路径参数中的id
	id := strings.TrimPrefix(c.Param("id"), "/")

	matchedHook := matchLoadedHook(id)
	if matchedHook == nil {
		c.String(http.StatusNotFound, "Hook not found.")
		return
	}

	// Check for allowed methods
	var allowedMethod bool

	switch {
	case len(matchedHook.HTTPMethods) != 0:
		for i := range matchedHook.HTTPMethods {
			if c.Request.Method == strings.ToUpper(strings.TrimSpace(matchedHook.HTTPMethods[i])) {
				allowedMethod = true
				break
			}
		}
	case *httpMethods != "":
		for _, v := range strings.Split(*httpMethods, ",") {
			if c.Request.Method == v {
				allowedMethod = true
				break
			}
		}
	default:
		allowedMethod = true
	}

	if !allowedMethod {
		c.String(http.StatusMethodNotAllowed, "")
		log.Printf("[%s] HTTP %s method not allowed for hook %q", req.ID, c.Request.Method, id)
		return
	}

	log.Printf("[%s] %s got matched\n", req.ID, id)

	for _, responseHeader := range responseHeaders {
		c.Header(responseHeader.Name, responseHeader.Value)
	}

	var err error

	// set contentType to IncomingPayloadContentType or header value
	req.ContentType = c.Request.Header.Get("Content-Type")
	if len(matchedHook.IncomingPayloadContentType) != 0 {
		req.ContentType = matchedHook.IncomingPayloadContentType
	}

	isMultipart := strings.HasPrefix(req.ContentType, "multipart/form-data;")

	if !isMultipart {
		req.Body, err = io.ReadAll(c.Request.Body)
		if err != nil {
			log.Printf("[%s] error reading the request body: %+v\n", req.ID, err)
		} else if *debug && len(req.Body) > 0 {
			// debug模式下输出请求体内容（限制长度避免日志过长）
			bodyStr := string(req.Body)
			if len(bodyStr) > 500 {
				bodyStr = bodyStr[:500] + "... (truncated)"
			}
			log.Printf("[%s] Request Body: %s", req.ID, bodyStr)
		}
	}

	req.ParseHeaders(c.Request.Header)
	req.ParseQuery(c.Request.URL.Query())

	switch {
	case strings.Contains(req.ContentType, "json"):
		err = req.ParseJSONPayload()
		if err != nil {
			log.Printf("[%s] %s", req.ID, err)
		}

	case strings.Contains(req.ContentType, "x-www-form-urlencoded"):
		err = req.ParseFormPayload()
		if err != nil {
			log.Printf("[%s] %s", req.ID, err)
		}

	case strings.Contains(req.ContentType, "xml"):
		err = req.ParseXMLPayload()
		if err != nil {
			log.Printf("[%s] %s", req.ID, err)
		}

	case isMultipart:
		err = c.Request.ParseMultipartForm(*maxMultipartMem)
		if err != nil {
			msg := fmt.Sprintf("[%s] error parsing multipart form: %+v\n", req.ID, err)
			log.Println(msg)
			c.String(http.StatusInternalServerError, "Error occurred while parsing multipart form.")
			return
		}

		for k, v := range c.Request.MultipartForm.Value {
			log.Printf("[%s] found multipart form value %q", req.ID, k)

			if req.Payload == nil {
				req.Payload = make(map[string]interface{})
			}

			req.Payload[k] = v[0]
		}

		for k, v := range c.Request.MultipartForm.File {
			var parseAsJSON bool
			for _, j := range matchedHook.JSONStringParameters {
				if j.Source == "payload" && j.Name == k {
					parseAsJSON = true
					break
				}
			}

			if !parseAsJSON && len(v[0].Header["Content-Type"]) > 0 {
				for _, j := range v[0].Header["Content-Type"] {
					if j == "application/json" {
						parseAsJSON = true
						break
					}
				}
			}

			if parseAsJSON {
				log.Printf("[%s] parsing multipart form file %q as JSON\n", req.ID, k)

				f, err := v[0].Open()
				if err != nil {
					msg := fmt.Sprintf("[%s] error parsing multipart form file: %+v\n", req.ID, err)
					log.Println(msg)
					c.String(http.StatusInternalServerError, "Error occurred while parsing multipart form file.")
					return
				}

				decoder := json.NewDecoder(f)
				decoder.UseNumber()

				var part map[string]interface{}
				err = decoder.Decode(&part)
				if err != nil {
					log.Printf("[%s] error parsing JSON payload file: %+v\n", req.ID, err)
				}

				if req.Payload == nil {
					req.Payload = make(map[string]interface{})
				}
				req.Payload[k] = part
			}
		}

	default:
		log.Printf("[%s] error parsing body payload due to unsupported content type header: %s\n", req.ID, req.ContentType)
	}

	// handle hook
	errors := matchedHook.ParseJSONParameters(req)
	for _, err := range errors {
		log.Printf("[%s] error parsing JSON parameters: %s\n", req.ID, err)
	}

	var ok bool

	if matchedHook.TriggerRule == nil {
		ok = true
	} else {
		req.AllowSignatureErrors = matchedHook.TriggerSignatureSoftFailures

		ok, err = matchedHook.TriggerRule.Evaluate(req)
		if err != nil {
			if !hook.IsParameterNodeError(err) {
				msg := fmt.Sprintf("[%s] error evaluating hook: %s", req.ID, err)
				log.Println(msg)
				c.String(http.StatusInternalServerError, "Error occurred while evaluating hook rules.")
				return
			}

			log.Printf("[%s] %v", req.ID, err)
		}
	}

	if ok {
		log.Printf("[%s] %s hook triggered successfully\n", req.ID, matchedHook.ID)

		for _, responseHeader := range matchedHook.ResponseHeaders {
			c.Header(responseHeader.Name, responseHeader.Value)
		}

		if matchedHook.CaptureCommandOutput {
			response, err := handleHook(matchedHook, req)

			if err != nil {
				if matchedHook.CaptureCommandOutputOnError {
					c.String(http.StatusInternalServerError, response)
				} else {
					c.Header("Content-Type", "text/plain; charset=utf-8")
					c.String(http.StatusInternalServerError, "Error occurred while executing the hook's command. Please check your logs for more details.")
				}
			} else {
				if matchedHook.SuccessHttpResponseCode != 0 {
					c.String(matchedHook.SuccessHttpResponseCode, response)
				} else {
					c.String(http.StatusOK, response)
				}
			}
		} else {
			if *verbose {
				log.Printf("[%s] executing hook in background\n", req.ID)
			}
			go func() {
				_, err := handleHook(matchedHook, req)
				if err != nil && *verbose {
					log.Printf("[%s] background hook execution failed: %v\n", req.ID, err)
				}
			}()

			if matchedHook.SuccessHttpResponseCode != 0 {
				c.String(matchedHook.SuccessHttpResponseCode, matchedHook.ResponseMessage)
			} else {
				c.String(http.StatusOK, matchedHook.ResponseMessage)
			}
		}

		return
	}

	// Check if a return code is configured for the hook
	if matchedHook.TriggerRuleMismatchHttpResponseCode != 0 {
		// 验证HTTP状态码是否有效（100-599范围）
		statusCode := matchedHook.TriggerRuleMismatchHttpResponseCode
		if statusCode < 100 || statusCode > 599 {
			// 无效的HTTP状态码，使用默认的200
			statusCode = http.StatusOK
		}
		c.String(statusCode, "Hook rules were not satisfied.")
	} else {
		c.String(http.StatusOK, "Hook rules were not satisfied.")
	}

	log.Printf("[%s] %s got matched, but didn't get triggered because the trigger rules were not satisfied\n", req.ID, matchedHook.ID)
}

func handleHook(h *hook.Hook, r *hook.Request) (string, error) {
	var errors []error

	// check the command exists
	var lookpath string
	if filepath.IsAbs(h.ExecuteCommand) || h.CommandWorkingDirectory == "" {
		lookpath = h.ExecuteCommand
	} else {
		lookpath = filepath.Join(h.CommandWorkingDirectory, h.ExecuteCommand)
	}

	cmdPath, err := exec.LookPath(lookpath)
	if err != nil {
		log.Printf("[%s] error in %s", r.ID, err)

		// check if parameters specified in execute-command by mistake
		if strings.IndexByte(h.ExecuteCommand, ' ') != -1 {
			s := strings.Fields(h.ExecuteCommand)[0]
			log.Printf("[%s] use 'pass-arguments-to-command' to specify args for '%s'", r.ID, s)
		}

		return "", err
	}

	cmd := exec.Command(cmdPath)
	cmd.Dir = h.CommandWorkingDirectory

	cmd.Args, errors = h.ExtractCommandArguments(r)
	for _, err := range errors {
		log.Printf("[%s] error extracting command arguments: %s\n", r.ID, err)
	}

	var envs []string
	envs, errors = h.ExtractCommandArgumentsForEnv(r)

	for _, err := range errors {
		log.Printf("[%s] error extracting command arguments for environment: %s\n", r.ID, err)
	}

	files, errors := h.ExtractCommandArgumentsForFile(r)

	for _, err := range errors {
		log.Printf("[%s] error extracting command arguments for file: %s\n", r.ID, err)
	}

	for i := range files {
		tmpfile, err := os.CreateTemp(h.CommandWorkingDirectory, files[i].EnvName)
		if err != nil {
			log.Printf("[%s] error creating temp file [%s]", r.ID, err)
			continue
		}
		log.Printf("[%s] writing env %s file %s", r.ID, files[i].EnvName, tmpfile.Name())
		if _, err := tmpfile.Write(files[i].Data); err != nil {
			log.Printf("[%s] error writing file %s [%s]", r.ID, tmpfile.Name(), err)
			continue
		}
		if err := tmpfile.Close(); err != nil {
			log.Printf("[%s] error closing file %s [%s]", r.ID, tmpfile.Name(), err)
			continue
		}

		files[i].File = tmpfile
		envs = append(envs, files[i].EnvName+"="+tmpfile.Name())
	}

	cmd.Env = append(os.Environ(), envs...)

	log.Printf("[%s] executing %s (%s) with arguments %q and environment %s using %s as cwd\n", r.ID, h.ExecuteCommand, cmd.Path, cmd.Args, envs, cmd.Dir)

	out, err := cmd.CombinedOutput()

	log.Printf("[%s] command output: %s\n", r.ID, out)

	if err != nil {
		log.Printf("[%s] error occurred: %+v\n", r.ID, err)
	}

	for i := range files {
		if files[i].File != nil {
			log.Printf("[%s] removing file %s\n", r.ID, files[i].File.Name())
			err := os.Remove(files[i].File.Name())
			if err != nil {
				log.Printf("[%s] error removing file %s [%s]", r.ID, files[i].File.Name(), err)
			}
		}
	}

	log.Printf("[%s] finished handling %s\n", r.ID, h.ID)

	// 推送WebSocket消息通知hook执行完成
	wsMessage := stream.Message{
		Type:      "hook_triggered",
		Timestamp: time.Now(),
		Data: stream.HookTriggeredMessage{
			HookID:   h.ID,
			HookName: h.ID,
			Method: func() string {
				if r.RawRequest != nil {
					return r.RawRequest.Method
				}
				return ""
			}(),
			RemoteAddr: func() string {
				if r.RawRequest != nil {
					return r.RawRequest.RemoteAddr
				}
				return ""
			}(),
			Success: err == nil,
			Output:  string(out),
			Error: func() string {
				if err != nil {
					return err.Error()
				} else {
					return ""
				}
			}(),
		},
	}
	stream.Global.Broadcast(wsMessage)

	return string(out), err
}

func reloadHooks(hooksFilePath string) {
	if router.HookManager != nil {
		if err := router.HookManager.ReloadHooks(hooksFilePath); err != nil {
			log.Printf("failed to reload hooks from %s: %v", hooksFilePath, err)
		}
		return
	}

	// 回退到原有逻辑
	log.Printf("reloading hooks from %s\n", hooksFilePath)

	newHooks := hook.Hooks{}

	err := newHooks.LoadFromFile(hooksFilePath, *asTemplate)

	if err != nil {
		log.Printf("couldn't load hooks from file! %+v\n", err)
	} else {
		seenHooksIds := make(map[string]bool)

		log.Printf("found %d hook(s) in file\n", len(newHooks))

		for _, hook := range newHooks {
			wasHookIDAlreadyLoaded := false

			for _, loadedHook := range loadedHooksFromFiles[hooksFilePath] {
				if loadedHook.ID == hook.ID {
					wasHookIDAlreadyLoaded = true
					break
				}
			}

			if (matchLoadedHook(hook.ID) != nil && !wasHookIDAlreadyLoaded) || seenHooksIds[hook.ID] {
				log.Printf("error: hook with the id %s has already been loaded!\nplease check your hooks file for duplicate hooks ids!", hook.ID)
				log.Println("reverting hooks back to the previous configuration")
				return
			}

			seenHooksIds[hook.ID] = true
			log.Printf("\tloaded: %s\n", hook.ID)
		}

		loadedHooksFromFiles[hooksFilePath] = newHooks
	}
}

func ReloadAllHooks() {
	if router.HookManager != nil {
		if err := router.HookManager.ReloadAllHooks(); err != nil {
			log.Printf("failed to reload all hooks: %v", err)
		}
	} else {
		// 回退到原有逻辑
		for _, hooksFilePath := range hooksFiles {
			reloadHooks(hooksFilePath)
		}
	}
}

func removeHooks(hooksFilePath string) {
	if router.HookManager != nil {
		router.HookManager.RemoveHooks(hooksFilePath)

		// 从hooksFiles列表中移除文件路径
		newHooksFiles := hooksFiles[:0]
		for _, filePath := range hooksFiles {
			if filePath != hooksFilePath {
				newHooksFiles = append(newHooksFiles, filePath)
			}
		}
		hooksFiles = newHooksFiles

		// 更新HookManager中的文件列表
		router.HookManager.HooksFiles = hooksFiles

		if !*verbose && !*noPanic && router.HookManager.GetHookCount() == 0 {
			log.SetOutput(os.Stdout)
			log.Fatalln("couldn't load any hooks from file!\naborting webhook execution since the -verbose flag is set to false.\nIf, for some reason, you want webhook to run without the hooks, either use -verbose flag, or -nopanic")
		}
		return
	}

	// 回退到原有逻辑
	log.Printf("removing hooks from %s\n", hooksFilePath)

	for _, hook := range loadedHooksFromFiles[hooksFilePath] {
		log.Printf("\tremoving: %s\n", hook.ID)
	}

	newHooksFiles := hooksFiles[:0]
	for _, filePath := range hooksFiles {
		if filePath != hooksFilePath {
			newHooksFiles = append(newHooksFiles, filePath)
		}
	}

	hooksFiles = newHooksFiles

	removedHooksCount := len(loadedHooksFromFiles[hooksFilePath])

	delete(loadedHooksFromFiles, hooksFilePath)

	log.Printf("removed %d hook(s) that were loaded from file %s\n", removedHooksCount, hooksFilePath)

	if !*verbose && !*noPanic && lenLoadedHooks() == 0 {
		log.SetOutput(os.Stdout)
		log.Fatalln("couldn't load any hooks from file!\naborting webhook execution since the -verbose flag is set to false.\nIf, for some reason, you want webhook to run without the hooks, either use -verbose flag, or -nopanic")
	}
}

func watchForFileChange() {
	for {
		select {
		case event := <-(*watcher).Events:
			if event.Op&fsnotify.Write == fsnotify.Write {
				log.Printf("hooks file %s modified\n", event.Name)
				reloadHooks(event.Name)
			} else if event.Op&fsnotify.Remove == fsnotify.Remove {
				if _, err := os.Stat(event.Name); os.IsNotExist(err) {
					log.Printf("hooks file %s removed, no longer watching this file for changes, removing hooks that were loaded from it\n", event.Name)
					if err := (*watcher).Remove(event.Name); err != nil {
						log.Printf("Error removing watcher for %s: %v\n", event.Name, err)
					}
					removeHooks(event.Name)
				}
			} else if event.Op&fsnotify.Rename == fsnotify.Rename {
				time.Sleep(100 * time.Millisecond)
				if _, err := os.Stat(event.Name); os.IsNotExist(err) {
					// file was removed
					log.Printf("hooks file %s removed, no longer watching this file for changes, and removing hooks that were loaded from it\n", event.Name)
					if err := (*watcher).Remove(event.Name); err != nil {
						log.Printf("Error removing watcher for %s: %v\n", event.Name, err)
					}
					removeHooks(event.Name)
				} else {
					// file was overwritten
					log.Printf("hooks file %s overwritten\n", event.Name)
					reloadHooks(event.Name)
					if err := (*watcher).Remove(event.Name); err != nil {
						log.Printf("Error removing watcher for %s: %v\n", event.Name, err)
					}
					if err := (*watcher).Add(event.Name); err != nil {
						log.Printf("Error adding watcher for %s: %v\n", event.Name, err)
					}
				}
			}
		case err := <-(*watcher).Errors:
			log.Println("watcher error:", err)
		}
	}
}

// makeHumanPattern builds a human-friendly URL for display.
func makeHumanPattern(prefix *string) string {
	if prefix == nil || *prefix == "" {
		return "/{id}"
	}
	return "/" + *prefix + "/{id}"
}
