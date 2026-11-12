package providerhttp

import (
 "context"
 "encoding/json"
 "net/http"
 "net/http/httptest"
 "strings"
 "testing"

 "github.com/Ploos-AS/Ploos-AI-Video-Connector/internal/jobs"
)
func TestGenerate(t *testing.T){
 server:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
  if r.URL.Path!="/generate" || r.Method!="POST" || r.Header.Get("Authorization")!="Bearer secret"{t.Errorf("bad request");w.WriteHeader(403);return}
  var request jobs.Request
  if err:=json.NewDecoder(r.Body).Decode(&request);err!=nil || request.Model!="demo" || request.Prompt!="test"{t.Errorf("bad body: %+v %v",request,err)}
  _,_=w.Write([]byte(`{"url":"https://example.invalid/video.mp4","mime_type":"video/mp4"}`))
 }))
 defer server.Close()
 p,err:=New(server.URL+"/generate","secret")
 if err!=nil{t.Fatal(err)}
 result,err:=p.Generate(context.Background(),jobs.Request{Model:"demo",Prompt:"test"})
 if err!=nil || result.MIMEType!="video/mp4"{t.Fatalf("result: %+v %v",result,err)}
}
func TestRejectUnsafeEndpoints(t *testing.T){
 for _,endpoint:=range []string{"https://127.0.0.1:1234/generate","http://example.com:1234/generate","http://8.8.8.8:1234/generate","http://127.0.0.1:1234/other","http://127.0.0.1:1234/generate?x=1","http://user@127.0.0.1:1234/generate","http://127.0.0.1/generate"}{
  if _,err:=New(endpoint,"secret");err==nil{t.Errorf("accepted %q",endpoint)}
 }
 if _,err:=New("http://127.0.0.1:1234/generate","");err==nil{t.Fatal("accepted empty token")}
}
func TestRejectRedirectAndOversizedResponse(t *testing.T){
 server:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){http.Redirect(w,r,"http://8.8.8.8/",302)}))
 defer server.Close()
 p,err:=New(server.URL+"/generate","secret")
 if err!=nil{t.Fatal(err)}
 if _,err=p.Generate(context.Background(),jobs.Request{Model:"demo",Prompt:"test"});err==nil{t.Fatal("redirect accepted")}
 server2:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){_,_=w.Write([]byte(strings.Repeat("x",maxResponse+1)))}))
 defer server2.Close()
 p,err=New(server2.URL+"/generate","secret")
 if err!=nil{t.Fatal(err)}
 if _,err=p.Generate(context.Background(),jobs.Request{Model:"demo",Prompt:"test"});err==nil{t.Fatal("oversized response accepted")}
}
