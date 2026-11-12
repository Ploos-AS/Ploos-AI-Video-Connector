package providerasync

import (
 "context"
 "net/http"
 "net/http/httptest"
 "testing"
 "time"

 "github.com/Ploos-AS/Ploos-AI-Video-Connector/internal/jobs"
)

func TestIdempotencyHeader(t *testing.T){
 const key="0123456789abcdef0123456789abcdef"
 seen:=false
 server:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
  if r.Method=="POST"{
   if r.Header.Get("Idempotency-Key")!=key{t.Errorf("missing key: %q",r.Header.Get("Idempotency-Key"))}
   seen=true
   _,_=w.Write([]byte(`{"id":"remote-1","status":"completed","url":"https://example.invalid/v.mp4","mime_type":"video/mp4"}`))
   return
  }
  t.Errorf("unexpected request %s",r.Method)
 }))
 defer server.Close()
 p,err:=New(server.URL+"/jobs","secret",100*time.Millisecond)
 if err!=nil{t.Fatal(err)}
 recorded:=""
 _,err=p.GenerateIdempotent(context.Background(),jobs.Request{Model:"demo",Prompt:"test"},key,func(id string)error{recorded=id;return nil})
 if err!=nil{t.Fatal(err)}
 if !seen || recorded!="remote-1"{t.Fatalf("seen=%v recorded=%s",seen,recorded)}
}
func TestRejectInvalidIdempotencyKey(t *testing.T){
 p,err:=New("http://127.0.0.1:9000/jobs","secret",time.Second)
 if err!=nil{t.Fatal(err)}
 for _,key:=range []string{"", "wrong", "0123456789abcdef0123456789abcdeG"}{
  if _,err:=p.GenerateIdempotent(context.Background(),jobs.Request{Model:"demo",Prompt:"test"},key,func(string)error{return nil});err==nil{t.Fatalf("accepted %q",key)}
 }
}
