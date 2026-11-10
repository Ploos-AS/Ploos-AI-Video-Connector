package main

import (
 "net/http"
 "net/http/httptest"
 "strings"
 "testing"
)

func TestAuthAndTools(t *testing.T){
 h:=handler(strings.Repeat("x",40))
 req:=httptest.NewRequest(http.MethodPost,"/mcp",strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`))
 w:=httptest.NewRecorder();h.ServeHTTP(w,req)
 if w.Code!=401 {t.Fatalf("missing auth: %d",w.Code)}
 req.Header.Set("Authorization","Bearer "+strings.Repeat("x",40))
 w=httptest.NewRecorder();h.ServeHTTP(w,req)
 if w.Code!=200 || !strings.Contains(w.Body.String(),"video_capabilities"){t.Fatalf("unexpected: %d %s",w.Code,w.Body.String())}
}
func TestRejectBrowserOrigin(t *testing.T){
 h:=handler(strings.Repeat("x",40))
 req:=httptest.NewRequest("POST","/mcp",strings.NewReader("{}"))
 req.Header.Set("Origin","https://evil.example")
 req.Header.Set("Authorization","Bearer "+strings.Repeat("x",40))
 w:=httptest.NewRecorder();h.ServeHTTP(w,req)
 if w.Code!=403 {t.Fatalf("origin accepted: %d",w.Code)}
}
