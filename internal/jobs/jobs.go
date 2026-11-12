// Package jobs provides bounded, provider-neutral asynchronous video jobs.
package jobs

import (
 "context"
 "crypto/rand"
 "encoding/hex"
 "errors"
 "sync"
 "time"
)

var ErrNotFound = errors.New("job not found")
var ErrBusy = errors.New("job capacity reached")
var ErrUnavailable = errors.New("video provider unavailable")
var ErrInvalid = errors.New("invalid video request")

type Request struct {
 Model string `json:"model"`
 Prompt string `json:"prompt"`
}
type Result struct {
 URL string `json:"url,omitempty"`
 MIMEType string `json:"mime_type,omitempty"`
}
type Job struct {
 ID string `json:"id"`
 Status string `json:"status"`
 Model string `json:"model"`
 Result *Result `json:"result,omitempty"`
 Error string `json:"error,omitempty"`
 CreatedAt time.Time `json:"created_at"`
 UpdatedAt time.Time `json:"updated_at"`
}
type Provider interface {
 Generate(context.Context, Request) (Result,error)
}
type Manager struct {
 mu sync.Mutex
 jobs map[string]*Job
 cancel map[string]context.CancelFunc
 provider Provider
 slots chan struct{}
 timeout time.Duration
 maxJobs int
 store Store
}
func New(provider Provider, concurrency, maxJobs int, timeout time.Duration) (*Manager,error) {
 if concurrency<1 || maxJobs<1 || timeout<=0 {return nil,ErrInvalid}
 return &Manager{jobs:make(map[string]*Job),cancel:make(map[string]context.CancelFunc),provider:provider,slots:make(chan struct{},concurrency),timeout:timeout,maxJobs:maxJobs},nil
}
// AttachStore loads persisted jobs and marks interrupted work as failed.
// Reconciliation of remote provider jobs is not yet supported.
func (m *Manager) AttachStore(store Store) error {
 if store==nil{return ErrInvalid}
 existing,err:=store.Load()
 if err!=nil{return err}
 m.mu.Lock()
 defer m.mu.Unlock()
 if len(m.jobs)!=0{return ErrInvalid}
 now:=time.Now().UTC()
 for _,job:=range existing{
  if job.Status=="queued" || job.Status=="running"{
   job.Status="failed"
   job.Error="connector restarted before job completed"
   job.UpdatedAt=now
   if err:=store.Save(job);err!=nil{return err}
  }
  copy:=job
  m.jobs[job.ID]=&copy
 }
 m.store=store
 return nil
}
func (m *Manager) Submit(req Request) (Job,error) {
 if req.Model=="" || req.Prompt=="" || len(req.Prompt)>4096 || len(req.Model)>128 {return Job{},ErrInvalid}
 if m.provider==nil {return Job{},ErrUnavailable}
 m.mu.Lock()
 defer m.mu.Unlock()
 if len(m.jobs)>=m.maxJobs {return Job{},ErrBusy}
 select {case m.slots<-struct{}{}:default:return Job{},ErrBusy}
 var bytes [16]byte
 if _,err:=rand.Read(bytes[:]);err!=nil {<-m.slots;return Job{},err}
 id:=hex.EncodeToString(bytes[:])
 now:=time.Now().UTC()
 job:=&Job{ID:id,Status:"queued",Model:req.Model,CreatedAt:now,UpdatedAt:now}
 ctx,cancel:=context.WithTimeout(context.Background(),m.timeout)
 m.jobs[id]=job
 m.cancel[id]=cancel
 if m.store!=nil {if err:=m.store.Save(*job);err!=nil {delete(m.jobs,id);delete(m.cancel,id);cancel();<-m.slots;return Job{},err}}
 go m.run(ctx,id,req)
 return *job,nil
}
func (m *Manager) run(ctx context.Context,id string,req Request) {
 defer func(){<-m.slots}()
 m.mu.Lock()
 if m.jobs[id].Status=="cancelled" {m.mu.Unlock();return}
 m.jobs[id].Status="running"
 m.jobs[id].UpdatedAt=time.Now().UTC()
 if m.store!=nil {_=m.store.Save(*m.jobs[id])}
 m.mu.Unlock()
 result,err:=m.provider.Generate(ctx,req)
 m.mu.Lock()
 defer m.mu.Unlock()
 job:=m.jobs[id]
 if job.Status!="cancelled" {
  if err!=nil {job.Status="failed";job.Error="video generation failed"
  } else if ctx.Err()!=nil {job.Status="failed";job.Error="video generation timed out"
  } else {job.Status="completed";job.Result=&result}
  job.UpdatedAt=time.Now().UTC()
 }
 if m.store!=nil {_=m.store.Save(*job)}
 m.cancel[id]()
 delete(m.cancel,id)
}
func (m *Manager) Get(id string) (Job,error) {
 m.mu.Lock();defer m.mu.Unlock()
 job,ok:=m.jobs[id];if !ok {return Job{},ErrNotFound}
 return *job,nil
}
func (m *Manager) Cancel(id string) (Job,error) {
 m.mu.Lock();defer m.mu.Unlock()
 job,ok:=m.jobs[id];if !ok {return Job{},ErrNotFound}
 if job.Status=="queued" || job.Status=="running" {
  job.Status="cancelled";job.UpdatedAt=time.Now().UTC()
  if m.store!=nil {if err:=m.store.Save(*job);err!=nil{return Job{},err}}
  m.cancel[id]()
 }
 return *job,nil
}
