package providerregistry

import (
 "context"
 "errors"
 "testing"

 "github.com/Ploos-AS/Ploos-AI-Video-Connector/internal/jobs"
)
type fakeProvider struct{called int}
func(p *fakeProvider)Generate(_ context.Context,_ jobs.Request)(jobs.Result,error){p.called++;return jobs.Result{URL:"https://example.invalid/v.mp4"},nil}
func TestExplicitRouting(t *testing.T){
 a,b:=&fakeProvider{},&fakeProvider{}
 r,err:=New([]Entry{{Name:"local",Models:[]string{"local-model"},Local:true,Provider:a},{Name:"paid",Models:[]string{"paid-model"},Provider:b}})
 if err!=nil{t.Fatal(err)}
 if _,err:=r.Generate(context.Background(),jobs.Request{Model:"local-model"});err!=nil{t.Fatal(err)}
 if a.called!=1 || b.called!=0{t.Fatalf("unexpected routing: local=%d paid=%d",a.called,b.called)}
 if _,err:=r.Generate(context.Background(),jobs.Request{Model:"unknown"});!errors.Is(err,jobs.ErrUnavailable){t.Fatalf("unknown model must fail closed: %v",err)}
 if b.called!=0{t.Fatal("paid fallback occurred")}
 models:=r.Models()
 if len(models)!=2 || models[0].ID!="local-model" || !models[0].Local{t.Fatalf("unexpected models: %+v",models)}
}
func TestRejectDuplicateModels(t *testing.T){
 p:=&fakeProvider{}
 _,err:=New([]Entry{{Name:"a",Models:[]string{"shared"},Provider:p},{Name:"b",Models:[]string{"shared"},Provider:p}})
 if !errors.Is(err,ErrConfiguration){t.Fatalf("duplicate model accepted: %v",err)}
}
