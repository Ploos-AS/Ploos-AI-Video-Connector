package jobs

import (
 "context"
 "errors"
 "path/filepath"
 "testing"
 "time"
)

type quickProvider struct{}
func(quickProvider)Generate(context.Context,Request)(Result,error){return Result{URL:"https://example.invalid/v.mp4",MIMEType:"video/mp4"},nil}

func TestTerminalJobsFreeCapacity(t *testing.T){
 m,err:=New(quickProvider{},1,1,time.Second)
 if err!=nil{t.Fatal(err)}
 first,err:=m.Submit(Request{Model:"demo",Prompt:"first"})
 if err!=nil{t.Fatal(err)}
 deadline:=time.Now().Add(time.Second)
 for{
  job,err:=m.Get(first.ID)
  if err!=nil{t.Fatal(err)}
  if job.Status=="completed"{break}
  if time.Now().After(deadline){t.Fatal("first job did not complete")}
  time.Sleep(time.Millisecond)
 }
 // Allow the worker to release its concurrency slot.
 var second Job
 for time.Now().Before(deadline){
  second,err=m.Submit(Request{Model:"demo",Prompt:"second"})
  if err==nil{break}
  if !errors.Is(err,ErrBusy){t.Fatal(err)}
  time.Sleep(time.Millisecond)
 }
 if err!=nil{t.Fatal(err)}
 if second.ID==first.ID{t.Fatal("job ID reused")}
 if _,err=m.Get(first.ID);!errors.Is(err,ErrNotFound){t.Fatalf("old job still present: %v",err)}
}

func TestPeriodicPrune(t *testing.T){
 m,err:=New(nil,1,10,time.Second)
 if err!=nil{t.Fatal(err)}
 store,err:=OpenSQLite(filepath.Join(t.TempDir(),"jobs.sqlite"))
 if err!=nil{t.Fatal(err)}
 defer store.Close()
 old:=time.Now().UTC().Add(-48*time.Hour)
 for _,j:=range []Job{{ID:"done",Status:"completed",UpdatedAt:old},{ID:"active",Status:"running",UpdatedAt:old}}{
  if err=store.Save(j);err!=nil{t.Fatal(err)}
 }
 if err=m.AttachStore(store);err!=nil{t.Fatal(err)}
 removed,err:=m.Prune(time.Now().UTC().Add(-24*time.Hour))
 if err!=nil{t.Fatal(err)}
 // AttachStore marks interrupted jobs failed with a fresh timestamp.
 if removed!=0{t.Fatalf("unexpected removal %d",removed)}
 if _,err=m.Get("done");!errors.Is(err,ErrNotFound){t.Fatal("expired completed job should be pruned at startup")}
}
