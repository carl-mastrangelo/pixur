package main

import (
	"reflect"

	"github.com/gopherjs/gopherjs/js"
)

var Ng *Angular = &Angular{js.Global.Get("angular")}

func main() {
	mod := Ng.Module("pixur", []string{"ngRoute"}, nil)

	mod.Config([]string{"$locationProvider"}, func(lp LocationProvider) {
		lp.SetHtml5Mode(true)
	})

	mod.Config([]string{"$routeProvider"}, func(rp RouteProvider) {
		rp.When("/", RouteObject{
			TemplateUrl:  "static/index_body.html",
			Controller:   "IndexCtrl",
			ControllerAs: "ctrl",
		})
		rp.When("/i/:picId?", RouteObject{
			TemplateUrl:  "static/index_body.html",
			Controller:   "IndexCtrl",
			ControllerAs: "ctrl",
		})
		rp.When("/p/:picId", RouteObject{
			TemplateUrl:  "static/viewer.html",
			Controller:   "ViewerCtrl",
			ControllerAs: "ctrl",
		})
		rp.When("/exp/qv/:picId", RouteObject{
			TemplateUrl:  "static/quickviewer.html",
			Controller:   "QuickViewerCtrl",
			ControllerAs: "ctrl",
		})
	})

	mod.Controller("IndexCtrl", js.Global.Get("IndexCtrl"))
	mod.Controller("ViewerCtrl", js.Global.Get("ViewerCtrl"))
	mod.Controller("QuickViewerCtrl", js.Global.Get("QuickViewerCtrl"))

	mod.Service("picsService", js.Global.Get("PicsService"))

	js.Global.Get("console").Call("log", mod)
}

type Angular struct {
	*js.Object
}

type NgModule struct {
	*js.Object
}

func (m *NgModule) Controller(name string, object interface{}) *NgModule {
	m.Call("controller", name, object)
	return m
}

func (m *NgModule) Service(name string, object interface{}) *NgModule {
	m.Call("service", name, object)
	return m
}

func (m *NgModule) Config(deps []string, configFn interface{}) {
	args := make([]interface{}, 0, len(deps)+1)
	for _, dep := range deps {
		args = append(args, dep)
	}
	args = append(args, configFn)
	m.Call("config", args)
}

type LocationProvider struct {
	*js.Object
}

type Html5ModeObject struct {
	Enabled      bool
	RequireBase  bool
	RewriteLinks bool
}

func (lp *LocationProvider) SetHtml5Mode(enable bool) {
	lp.Call("html5Mode", enable)
}

func (lp *LocationProvider) GetHtml5Mode() interface{} {
	return lp.Call("html5Mode")
}

type RouteProvider struct {
	*js.Object
}

func (rp *RouteProvider) When(path string, r RouteObject) {
	rp.Call("when", path, wrap(r))
}

type RouteObject struct {
	TemplateUrl  string `jss:"templateUrl"`
	Controller   string `jss:"controller"`
	ControllerAs string `jss:"controllerAs"`
}

func (a *Angular) Module(name string, requires []string, configFn func()) *NgModule {
	if requires == nil {
		requires = make([]string, 0)
	}

	return &NgModule{a.Call("module", name, requires, configFn)}
}

func wrap(val interface{}) map[string]interface{} {
	m := make(map[string]interface{})
	t := reflect.TypeOf(val)

	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
		// Use jss because js is reserved and not handled properly
		if name := sf.Tag.Get("jss"); name != "" {
			m[name] = reflect.ValueOf(val).Field(i).Interface()
		}
	}
	return m
}
