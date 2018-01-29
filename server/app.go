package main

import (
	"html/template"
	"io"
	"net/http"

	"github.com/itsjamie/go-bindata-templates"
	"github.com/gin-gonic/gin"
	"github.com/nu7hatch/gouuid"
	"github.com/olebedev/config"
	"github.com/elazarl/go-bindata-assetfs"
)

// App struct.
// There is no singleton anti-pattern,
// all variables defined locally inside
// this struct.
type App struct {
	Engine *gin.Engine
	Conf   *config.Config
	React  *React
	API    *API
}

// NewApp returns initialized struct
// of main server application.
func NewApp(opts ...AppOptions) *App {
	options := AppOptions{}
	for _, i := range opts {
		options = i
		break
	}

	options.init()

	// Parse config yaml string from ./conf.go
	conf, err := config.ParseYaml(confString)
	Must(err)

	// Set config variables delivered from main.go:11
	// Variables defined as ./conf.go:3
	conf.Set("debug", debug)
	conf.Set("commitHash", commitHash)

	// Parse environ variables for defined
	// in config constants
	conf.Env()

	// Make an engine
	engine := gin.New()

	// Logger for each request
	engine.Use(gin.Logger())

	// Recovery for any panics issue in system
	engine.Use(gin.Recovery())

	engine.GET("/favicon.ico", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/static/images/favicon.ico")
	})

	engine.LoadHTMLGlob("server/data/templates/*")


	// Initialize the application
	app := &App{
		Conf:   conf,
		Engine: engine,
		API:    &API{},
		React: NewReact(
			conf.UString("duktape.path"),
			conf.UBool("debug"),
			engine,
		),
	}

	// Map app and uuid for every requests
	app.Engine.Use(func(c *gin.Context) {
		c.Set("app", app)
		id, _ := uuid.NewV4()
		c.Set("uuid", id)
		c.Next()
	})

	// Bind api hadling for URL api.prefix
	app.API.Bind(
		app.Engine.Group(
			app.Conf.UString("api.prefix"),
		),
	)

	// Create file http server from bindata
	fileServerHandler := http.FileServer(&assetfs.AssetFS{
		Asset:     Asset,
		AssetDir:  AssetDir,
		AssetInfo: AssetInfo,
	})

	//Serve static via bindata and handle via react app
	//in case when static file was not found
	engine.NoRoute(func(c *gin.Context){

		if _, err := Asset(c.Request.URL.Path[1:]); err == nil {
			fileServerHandler.ServeHTTP(
				c.Writer,
				c.Request)
			return
		}

		app.React.Handle(c)
	})


	return app
}

// Run runs the app
func (app *App) Run() {
	Must(app.Engine.Run(":" + app.Conf.UString("port")))
}

// Template is custom renderer for Echo, to render html from bindata
type Template struct {
	templates *template.Template
}

// NewTemplate creates a new template
func NewTemplate() *Template {
	return &Template{
		templates: binhtml.New(Asset, AssetDir).MustLoadDirectory("templates"),
	}
}

// Render renders template
func (t *Template) Render(w io.Writer, name string, data interface{}, c gin.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

// AppOptions is options struct
type AppOptions struct{}

func (ao *AppOptions) init() { /* write your own*/ }
