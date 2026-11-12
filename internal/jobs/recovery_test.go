package jobs

import (
 "context"
 "path/filepath"
 "sync/atomic"
 "testing"
 "time"
)

type recoveryProvider struct{posts, resumes atomic.Int32}
func(p *recoveryProvider)Generate(ctx context.Context,req Request)(Result,error){return p.GenerateTracked(ctx,req,func(string)error{return nil})}
func(p *recoveryProvider)GenerateTracked(ctx context.Context,req Request,record func(string)error)(Result,error){
 p.posts.Add(1)
 if err:=record("remote-123");err!=nil{return Result{},err}
 <-ctx.Done()
 return Result{},ctx.Err()
}
func(p *recoveryProvider)Resume(ctx context.Context,id string)(Result,error){
 p.resumes.Add(1)
 if id!="remote-123"{return Result{},ErrInvalid}
 return Result{URL:"https://example.invalid/video.mp4",MIMEType:"video/mp4"},nil
}
func TestResumeWithoutResubmit(t *testing.T){
 store,err:=OpenSQLite(filepath.Join(t.TempDir(),"jobs.sqlite"))
 if err!=nil{t.Fatal(err)}
 defer store.Close()
 now:=time.Now().UTC()
 if err:=store.Save(Job{ID:"local-1",Status:"running",Model:"demo",RemoteID:"remote-123",CreatedAt:now,UpdatedAt:now});err!=nil{t.Fatal(err)}
 p:=&recoveryProvider{}
 m,err:=New(p,2,10,time.Second)
 if err!=nil{t.Fatal(err)}
 if err:=m.AttachStore(store);err!=nil{t.Fatal(err)}
 deadline:=time.Now().Add(time.Second)
 for time.Now().Before(deadline){
  job,err:=m.Get("local-1")
  if err!=nil{t.Fatal(err)}
  if job.Status=="completed"{
   if p.posts.Load()!=0 || p.resumes.Load()!=1{t.Fatalf("posts=%d resumes=%d",p.posts.Load(),p.resumes.Load())}
   return
  }
  time.Sleep(time.Millisecond)
 }
 t.Fatal("recovery did not finish")
}
func TestMissingRemoteIDFailsClosed(t *testing.T){
 store,err:=OpenSQLite(filepath.Join(t.TempDir(),"jobs.sqlite"))
 if err!=nil{t.Fatal(err)}
 defer store.Close()
 now:=time.Now().UTC()
 if err:=store.Save(Job{ID:"local-2",Status:"running",CreatedAt:now,UpdatedAt:now});err!=nil{t.Fatal(err)}
 p:=&recoveryProvider{}
 m,_:=New(p,2,10,time.Second)
 if err:=m.AttachStore(store);err!=nil{t.Fatal(err)}
 job,_:=m.Get("local-2")
 if job.Status!="failed" || p.posts.Load()!=0{t.Fatalf("unsafe recovery: %+v",job)}
}
