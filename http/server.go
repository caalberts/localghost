package http

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
	"sync"

	"github.com/caalberts/localroast/types"
	log "github.com/sirupsen/logrus"
)

// Server interface.
type Server interface {
	ListenAndServe() error
	Watch(chan []types.Schema)
}

type server struct {
	*http.Server
	router *router
}

// ServerFunc is a constructor for a new server.
type ServerFunc func(port string) Server

// NewServer creates a http server running on given port with handlers based on given schema.
func NewServer(port string) Server {
	router := newRouter()

	return &server{
		Server: &http.Server{
			Addr:    ":" + port,
			Handler: router,
		},
		router: router,
	}
}

func (s *server) Watch(incomingSchemas chan []types.Schema) {
	go func() {
		for {
			schemas := <-incomingSchemas
			log.Info("updating router with new schema")
			s.router.updateSchema(schemas)
		}
	}()
}

type router struct {
	sync.Mutex
	http.Handler
}

func newRouter() *router {
	rtr := router{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotImplemented)
		}),
	}
	return &rtr
}

func (rtr *router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rtr.Lock()
	defer rtr.Unlock()
	rtr.Handler.ServeHTTP(w, r)
}

func (rtr *router) updateSchema(schemas []types.Schema) {
	rtr.Lock()
	defer rtr.Unlock()
	router := httprouter.New()
	for i := range schemas {
		router.Handle(schemas[i].Method, schemas[i].Path, handlerFunc(schemas[i]))
	}
	rtr.Handler = router
}

func handlerFunc(schema types.Schema) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		log.Infof("request: %s %s", r.Method, r.URL)
		log.Infof("response status: %d, body: %s", schema.Status, schema.Response)
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(schema.Status)
		w.Write(schema.Response)
	}
}
