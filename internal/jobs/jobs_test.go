package jobs

import (
 "context"
 "errors"
 "testing"
 "time"
)
type fakeProvider struct {fn func(context.Context,Request)(Result,error)}
func (f fakeProvider) Generate(ctx context.Context,r Request)(Result,error){return f.fn(ctx,r)}
func TestLifecycle(t *testing.T){
 m,err:=New(fakeProvider{func(ctx context.Context,r Request)(Result,error){return Result{URL:"https://example.invalid/video.mp4",MIMEType:"video/mp4"},nil}},1,4,time.Second)
 if err!=nil {t.Fatal(err)}
 j,err:=m.Submit(Request{Model:"demo",Prompt:"a test"})
 if err!=nil {t.Fatal(err)}
 deadline:=time.After(time.Second)
 for {
  got,err:=m.Get(j.ID);if err!=nil {t.Fatal(err)}
  if got.Status=="completed" {if got.Result==nil || got.Result.MIMEType!="video/mp4" {t.Fatalf("bad result: %+v",got)};break}
  select {case <-deadline:t.Fatal("job never completed");default:time.Sleep(time.Millisecond)}
 }
 if _,err:=m.Get("missing");!errors.Is(err,ErrNotFound){t.Fatalf("missing job: %v",err)}
}
func TestCancelAndCapacity(t *testing.T){
 started:=make(chan struct{})
 m,_:=New(fakeProvider{func(ctx context.Context,r Request)(Result,error){close(started);<-ctx.Done();return Result{},ctx.Err()}},1,1,time.Second)
 j,err:=m.Submit(Request{Model:"demo",Prompt:"test"});if err!=nil {t.Fatal(err)}
 <-started
 if _,err:=m.Submit(Request{Model:"demo",Prompt:"test"});!errors.Is(err,ErrBusy){t.Fatalf("capacity: %v",err)}
 got,err:=m.Cancel(j.ID);if err!=nil || got.Status!="cancelled"{t.Fatalf("cancel: %+v %v",got,err)}
 deadline:=time.After(time.Second)
 for {got,_=m.Get(j.ID);if got.Status!="cancelled"{t.Fatal("cancel overwritten")}
  select {case <-deadline:return;default:time.Sleep(time.Millisecond)}
 }
}
func TestFailClosed(t *testing.T){
 m,_:=New(nil,1,2,time.Second)
 if _,err:=m.Submit(Request{Model:"demo",Prompt:"test"});!errors.Is(err,ErrUnavailable){t.Fatal(err)}
 if _,err:=m.Submit(Request{});!errors.Is(err,ErrInvalid){t.Fatal(err)}
}
