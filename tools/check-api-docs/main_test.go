package main

import "testing"

func TestCollectRoutesTracksReassignedGroupsAndAuthentication(t *testing.T) {
	routes, err := collectRoutes("routes.go", `package routes
func register(eng *Engine) {
 routes := eng.Group("/instance")
 routes.Use(m.AuthAdmin)
 routes.GET("/info/:instanceId", handler)
 routes = eng.Group("/instance")
 routes.Use(m.Auth)
 routes.POST("/connect", handler)
 public := eng.Group("/public")
 public.GET("/status", handler)
 eng.GET("/server/ok", handler)
 eng.GET("/manager/*any", handler)
}`)
	if err != nil {
		t.Fatal(err)
	}
	want := map[route]string{
		{"get", "/instance/info/{param}"}: "AdminApiKey",
		{"post", "/instance/connect"}:     "InstanceApiKey",
		{"get", "/public/status"}:         "",
		{"get", "/server/ok"}:             "",
	}
	if len(routes) != len(want) {
		t.Fatalf("got %v; want %v", routes, want)
	}
	for key, security := range want {
		if got, ok := routes[key]; !ok || got != security {
			t.Errorf("%v: got %q (%v); want %q", key, got, ok, security)
		}
	}
}

func TestNormalizePostmanAndSwaggerPaths(t *testing.T) {
	for _, path := range []string{"{{host}}/instance/logs/:instanceId?limit=50", "/instance/logs/{instanceId}"} {
		if got := normalize(path); got != "/instance/logs/{param}" {
			t.Errorf("normalize(%q) = %q", path, got)
		}
	}
}
