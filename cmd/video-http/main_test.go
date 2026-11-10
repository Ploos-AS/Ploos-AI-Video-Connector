package main

import (
 "net/http"
 "net/http/httptest"
 "strings"
 "testing"
)

func request(method,body string)*http.Request{
 r:=httptest.NewRequest(method,"/mcp",strings.NewReader(body))
 r.Header.Set("Content-Type","application/json")
 r.Header.Set("Accept","application/json, text/event-stream")
 r.Header.Set("MCP-Protocol-Version",protocolVersion)
 r.Header.Set("Authorization","Bearer "+strings.Repeat("x",40))
 return r
}
func TestAuthAndTools(t *testing.T){
 h:=handler(strings.Repeat("x",40))
 req:=request("POST",`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`)
 req.Header.Del("Authorization")
 w:=httptest.NewRecorder();h.ServeHTTP(w,req)
 if w.Code!=401 {t.Fatalf("missing auth: %d",w.Code)}
 req.Header.Set("Authorization","Bearer "+strings.Repeat("x",40))
 w=httptest.NewRecorder();h.ServeHTTP(w,req)
 if w.Code!=200 || !strings.Contains(w.Body.String(),"video_capabilities"){t.Fatalf("unexpected: %d %s",w.Code,w.Body.String())}
}
func TestInitialize(t *testing.T){
 w:=httptest.NewRecorder()
 handler(strings.Repeat("x",40)).ServeHTTP(w,request("POST",`{"jsonrpc":"2.0","id":3,"method":"initialize","params":{}}`))
 if w.Code!=200 || !strings.Contains(w.Body.String(),protocolVersion){t.Fatalf("initialize: %d %s",w.Code,w.Body.String())}
}
func TestNotification(t *testing.T){
 w:=httptest.NewRecorder()
 handler(strings.Repeat("x",40)).ServeHTTP(w,request("POST",`{"jsonrpc":"2.0","method":"notifications/initialized"}`))
 if w.Code!=202 || w.Body.Len()!=0 {t.Fatalf("notification: %d %s",w.Code,w.Body.String())}
}
func TestRejectBrowserOrigin(t *testing.T){
 req:=request("POST",`{"jsonrpc":"2.0","id":1,"method":"ping"}`)
 req.Header.Set("Origin","https://evil.example")
 w:=httptest.NewRecorder();handler(strings.Repeat("x",40)).ServeHTTP(w,req)
 if w.Code!=403 {t.Fatalf("origin accepted: %d",w.Code)}
}
func TestNegotiation(t *testing.T){
 for _,tc:=range []struct{name,header,value string;status int}{
  {"wrong content type","Content-Type","text/plain",415},
  {"missing event stream accept","Accept","application/json",406},
  {"unsupported version","MCP-Protocol-Version","1900-01-01",400},
 }{
  t.Run(tc.name,func(t *testing.T){
   req:=request("POST",`{"jsonrpc":"2.0","id":1,"method":"ping"}`)
   req.Header.Set(tc.header,tc.value)
   w:=httptest.NewRecorder();handler(strings.Repeat("x",40)).ServeHTTP(w,req)
   if w.Code!=tc.status {t.Fatalf("got %d, want %d",w.Code,tc.status)}
  })
 }
}

func TestOAuthDiscoveryOptIn(t *testing.T){
 token:=strings.Repeat("x",40)
 path:="/.well-known/oauth-protected-resource"
 req:=httptest.NewRequest("GET",path,nil)
 w:=httptest.NewRecorder()
 handler(token).ServeHTTP(w,req)
 if w.Code!=404 {t.Fatalf("metadata enabled by default: %d",w.Code)}
 h:=handlerWithOAuthMetadata(token,"https://video.example","https://auth.example")
 w=httptest.NewRecorder();h.ServeHTTP(w,req)
 if w.Code!=200 || !strings.Contains(w.Body.String(),"authorization_servers") {t.Fatalf("metadata: %d %s",w.Code,w.Body.String())}
 unauthorized:=request("POST",`{"jsonrpc":"2.0","id":1,"method":"ping"}`)
 unauthorized.Header.Del("Authorization")
 w=httptest.NewRecorder();h.ServeHTTP(w,unauthorized)
 if w.Code!=401 || !strings.Contains(w.Header().Get("WWW-Authenticate"),"resource_metadata") {t.Fatalf("challenge: %d %s",w.Code,w.Header().Get("WWW-Authenticate"))}
}
func TestRejectInsecureOAuthMetadata(t *testing.T){
 if validHTTPS("http://auth.example") || validHTTPS("https://user:pass@auth.example") || validHTTPS("https://auth.example?x=1") {t.Fatal("accepted unsafe metadata URL")}
}
