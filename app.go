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
	"strings"
	"time"

	"github.com/mycoool/gohook/internal/config"
	"github.com/mycoool/gohook/internal/database"
	"github.com/mycoool/gohook/internal/pidfile"
	"github.com/mycoool/gohook/internal/webhook"

	"github.com/fsnotify/fsnotify"
	"github.com/gin-gonic/gin"
	"github.com/mycoool/gohook/internal/router"
	"github.com/mycoool/gohook/ui"
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

	responseHeaders webhook.ResponseHeaders
	hooksFiles      webhook.HooksFiles

	loadedHooksFromFiles = make(map[string]webhook.Hooks)

	watcher *fsnotify.Watcher
	signals chan os.Signal
	pidFile *pidfile.PIDFile
	setUID  = 0
	setGID  = 0
	socket  = ""
	addr    = ""
)

// version info
var vInfo = &ui.VersionInfo{
	Version:   Version,
	Commit:    Commit,
	BuildDate: BuildDate,
}

func main() {
	flag.Var(&hooksFiles, "hooks", "path to the json file containing defined hooks the webhook should serve, use multiple times to load from different files")
	flag.Var(&responseHeaders, "header", "response header to return, specified in format name=value, use multiple times to set multiple headers")

	// register platform-specific flags
	platformFlags()

	flag.Parse()

	if *justDisplayVersion {
		fmt.Println("gohook version " + Version)
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

	// set mode according to mode flag
	//types.GoHookAppConfig.SetMode(*mode)

	if len(hooksFiles) == 0 {
		hooksFiles = append(hooksFiles, "hooks.json")
	}

	// logQueue is a queue for log messages encountered during startup. We need
	// to queue the messages so that we can handle any privilege dropping and
	// log file opening prior to writing our first log message.
	var logQueue []string

	// set gin mode according to debug flag, must be set before InitRouter
	if *ginDebug {
		gin.SetMode(gin.DebugMode)
		log.Printf("running in debug mode")
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// first try to load app config to get port setting
	// create a temporary router instance to load config
	webhook.LoadedHooksFromFiles = &loadedHooksFromFiles
	webhook.HookManager = webhook.NewHookManager(&loadedHooksFromFiles, hooksFiles, *asTemplate)
	router.InitRouter() // this will load app.yaml config file

	// determine final port: app.yaml > command line flag > default
	finalPort := *port // use command line flag by default

	// check if app.yaml exists
	if _, err := os.Stat("app.yaml"); err == nil {
		log.SetPrefix("[GoHook] ")
		// app.yaml exists, use port from app.yaml
		if appConfig := config.GetAppConfig(); appConfig != nil {
			finalPort = appConfig.Port
			log.Printf("listen port %d", finalPort)
		} else {
			log.Printf("listen port %d", finalPort)
		}
	} else {
		// app.yaml not found, use command line flag
		log.Printf("using port %d from command line flag (no app.yaml found)", finalPort)
	}

	// by default the listen address is ip:port, but this may be modified by trySocketListener
	addr = fmt.Sprintf("%s:%d", *ip, finalPort)

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

	log.SetPrefix("[GoHook] ")
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

	log.Println("version " + Version + " starting")

	// set os signal watcher
	setupSignals()

	// load and parse hooks
	for _, hooksFilePath := range hooksFiles {
		log.Printf("attempting to load hooks from %s\n", hooksFilePath)

		newHooks := webhook.Hooks{}

		err := newHooks.LoadFromFile(hooksFilePath, *asTemplate)

		if err != nil {
			log.Printf("couldn't load hooks from file! %+v\n", err)
		} else {
			log.Printf("found %d hook(s) in file\n", len(newHooks))

			for _, hookValue := range newHooks {
				if webhook.HookManager.MatchLoadedHook(hookValue.ID) != nil {
					log.Fatalf("error: hook with the id %s has already been loaded!\nplease check your hooks file for duplicate hooks ids!\n", hookValue.ID)
				}
				log.Printf("\tloaded: %s\n", hookValue.ID)
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

	if !*verbose && !*noPanic && webhook.HookManager.LenLoadedHooks() == 0 {
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

		go webhook.WatchForFileChange(watcher, &loadedHooksFromFiles, hooksFiles, *asTemplate)
	}

	// gin mode has been set before

	// router has been initialized before, here just get the instance
	r := router.GetRouter()

	// register frontend UI router, this will handle root path "/"
	ui.Register(r, *vInfo, true)

	// enable method not allowed handling
	r.HandleMethodNotAllowed = true

	// set gin middleware
	if *debug {
		// debug mode use detailed log middleware
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
		// production mode use simple log
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

	// Initialize database
	if appConfig := config.GetAppConfig(); appConfig != nil {
		// Convert types.DatabaseConfig to database.DatabaseConfig
		dbConfig := &database.DatabaseConfig{
			Type:     appConfig.Database.Type,
			Database: appConfig.Database.Database,
			Host:     appConfig.Database.Host,
			Port:     appConfig.Database.Port,
			Username: appConfig.Database.Username,
			Password: appConfig.Database.Password,
		}

		// Initialize database connection
		err := database.InitDatabase(dbConfig)
		if err != nil {
			log.Printf("Failed to initialize database: %v", err)
			// Use default SQLite configuration as fallback
			defaultConfig := database.DefaultDatabaseConfig()
			if initErr := database.InitDatabase(defaultConfig); initErr != nil {
				log.Fatalf("Failed to initialize database with default config: %v", initErr)
			}
		}

		// Perform database migration
		if err := database.AutoMigrate(); err != nil {
			log.Printf("Failed to migrate database: %v", err)
		}

		// Initialize global log service
		database.InitLogService()

		// Start automatic log cleanup task
		retentionDays := appConfig.Database.LogRetentionDays
		if retentionDays <= 0 {
			retentionDays = 30 // default retention period
		}
		database.ScheduleLogCleanup(retentionDays)

		// Register log routes
		logRouter := router.NewLogRouter()
		logRouter.RegisterLogRoutes(r.Group(""))

		log.Println("Database initialized and log routes registered")
	}

	// Clean up input
	*httpMethods = strings.ToUpper(strings.ReplaceAll(*httpMethods, " ", ""))

	// note: root path "/" is now handled by frontend UI router (registered in router.InitRouter())

	// webhook router - supports all HTTP methods
	hooksPath := "/" + *hooksURLPrefix + "/*id"
	r.Any(hooksPath, ginHookHandler)

	// Create common HTTP server settings
	svr := &http.Server{
		Handler: r,
	}

	// Serve HTTP
	if !*secure {
		log.Printf("serving hooks on http://%s%s", addr, webhook.MakeHumanPattern(hooksURLPrefix))
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

	log.Printf("serving hooks on https://%s%s", addr, webhook.MakeHumanPattern(hooksURLPrefix))
	log.Print(svr.ServeTLS(ln, *cert, *key))
}

func ginHookHandler(c *gin.Context) {
	req := &webhook.Request{
		ID:         c.GetString("request-id"), // can be set by middleware
		RawRequest: c.Request,
	}

	// if there is no request-id, generate a simple ID
	if req.ID == "" {
		req.ID = fmt.Sprintf("%d", time.Now().UnixNano())
	}

	log.Printf("[%s] incoming HTTP %s request from %s\n", req.ID, c.Request.Method, c.Request.RemoteAddr)

	// debug mode output more request information
	if *debug {
		log.Printf("[%s] Request Headers: %v", req.ID, c.Request.Header)
		log.Printf("[%s] Request URL: %s", req.ID, c.Request.URL.String())
		log.Printf("[%s] User-Agent: %s", req.ID, c.Request.UserAgent())
	}

	// get id from path parameter
	id := strings.TrimPrefix(c.Param("id"), "/")

	matchedHook := webhook.HookManager.MatchLoadedHook(id)
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
			// debug mode output request body content (limit length to avoid log too long)
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
			if !webhook.IsParameterNodeError(err) {
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
			response, err := webhook.HandleHook(matchedHook, req)

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
				_, err := webhook.HandleHook(matchedHook, req)
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

	// check if a return code is configured for the hook
	if matchedHook.TriggerRuleMismatchHttpResponseCode != 0 {
		// validate HTTP status code is valid (100-599 range)
		statusCode := matchedHook.TriggerRuleMismatchHttpResponseCode
		if statusCode < 100 || statusCode > 599 {
			// invalid HTTP status code, use default 200
			statusCode = http.StatusOK
		}
		c.String(statusCode, "Hook rules were not satisfied.")
	} else {
		c.String(http.StatusOK, "Hook rules were not satisfied.")
	}

	log.Printf("[%s] %s got matched, but didn't get triggered because the trigger rules were not satisfied\n", req.ID, matchedHook.ID)
}
