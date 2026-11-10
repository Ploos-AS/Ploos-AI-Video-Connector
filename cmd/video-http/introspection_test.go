package main

import (
 "context"
 "fmt"
 "net/http"
 "net/http/httptest"
 "strings"
 "testing"
 "time"
)

func TestIntrospectionValidation(t *testing.T){
 endpoint:=httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
  if r.Method!="POST" || r.Header.Get("Authorization")=="" {t.Error("missing POST credentials")}
  _=r.ParseForm()
  active:=r.Form.Get("token")=="good"
  fmt.Fprintf(w,`{"active":%t,"iss":"https://issuer.example","aud":["https://video.example"],"exp":%d,"scope":"mcp:read other"}`,active,time.Now().Add(time.Hour).Unix())
 }))
 defer endpoint.Close()
 i,err:=newIntrospector(endpoint.URL,"client","secret","https://issuer.example","https://video.example","mcp:read")
 if err!=nil{t.Fatal(err)}
 i.client=endpoint.Client()
 if err:=i.validate(context.Background(),"good");err!=nil{t.Fatal(err)}
 if err:=i.validate(context.Background(),"bad");err==nil{t.Fatal("inactive token accepted")}
 i.scope="mcp:write"
 if err:=i.validate(context.Background(),"good");err==nil{t.Fatal("missing scope accepted")}
}
func TestOAuthHandlerRejectsStaticToken(t *testing.T){
 i:=&introspector{endpoint:"https://127.0.0.1:1/introspect",clientID:"id",clientSecret:"secret",issuer:"https://issuer.example",audience:"https://video.example",scope:"mcp:read",client:&http.Client{Timeout:100*time.Millisecond}}
 req:=request("POST",`{"jsonrpc":"2.0","id":1,"method":"ping"}`)
 req.Header.Set("Authorization","Bearer "+strings.Repeat("x",40))
 w:=httptest.NewRecorder()
 handlerWithValidator(strings.Repeat("x",40),"https://video.example","https://issuer.example",i).ServeHTTP(w,req)
 if w.Code!=401{t.Fatalf("static fallback accepted: %d",w.Code)}
}
func TestIntrospectionConfig(t *testing.T){
 if _,err:=newIntrospector("http://localhost/introspect","id","secret","https://issuer.example","https://video.example","mcp:read");err==nil {t.Fatal("insecure endpoint accepted")}
 if _,err:=newIntrospector("https://issuer.example/introspect","","secret","https://issuer.example","https://video.example","mcp:read");err==nil {t.Fatal("missing credentials accepted")}
}
