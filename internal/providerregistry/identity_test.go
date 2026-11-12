package providerregistry

import (
 "context"
 "errors"
 "testing"

 "github.com/Ploos-AS/Ploos-AI-Video-Connector/internal/jobs"
)
type resumeProbe struct{called int}
func(p *resumeProbe)Generate(context.Context,jobs.Request)(jobs.Result,error){return jobs.Result{},nil}
func(p *resumeProbe)GenerateTracked(ctx context.Context,req jobs.Request,record func(string)error)(jobs.Result,error){return jobs.Result{},nil}
func(p *resumeProbe)Resume(ctx context.Context,id string)(jobs.Result,error){p.called++;return jobs.Result{URL:"https://example.invalid/video.mp4"},nil}
func TestIdentityPreventsWrongProviderRecovery(t *testing.T){
 original:=&resumeProbe{}
 r1,err:=New([]Entry{{Name:"local",Identity:"provider-a",Models:[]string{"model-1"},Provider:original}})
 if err!=nil{t.Fatal(err)}
 identity,err:=r1.IdentityFor("model-1")
 if err!=nil || identity!="provider-a"{t.Fatalf("identity=%q err=%v",identity,err)}
 changed:=&resumeProbe{}
 r2,err:=New([]Entry{{Name:"local",Identity:"provider-b",Models:[]string{"model-1"},Provider:changed}})
 if err!=nil{t.Fatal(err)}
 if _,err:=r2.ResumeForIdentity(context.Background(),"model-1",identity,"remote-1");!errors.Is(err,jobs.ErrUnavailable){t.Fatalf("expected identity mismatch, got %v",err)}
 if changed.called!=0{t.Fatal("contacted wrong provider")}
 if _,err:=r1.ResumeForIdentity(context.Background(),"model-1",identity,"remote-1");err!=nil{t.Fatal(err)}
 if original.called!=1{t.Fatal("original provider not resumed")}
}
