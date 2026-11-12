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
var ErrStorage = errors.New("job storage failure")

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
 RemoteID string `json:"remote_id,omitempty"`
 ProviderID string `json:"provider_id,omitempty"`
 IdempotencyKey string `json:"idempotency_key,omitempty"`
 Result *Result `json:"result,omitempty"`
 Error string `json:"error,omitempty"`
 CreatedAt time.Time `json:"created_at"`
 UpdatedAt time.Time `json:"updated_at"`
}
type Provider interface {
 Generate(context.Context, Request) (Result,error)
}
// RecoverableProvider can resume polling a known remote job without submitting it again.
type RecoverableProvider interface {
 Provider
 GenerateTracked(context.Context, Request, func(string) error) (Result,error)
 Resume(context.Context,string) (Result,error)
}
// IdempotentProvider supports safe replay with a provider-enforced key.
type IdempotentProvider interface {
 RecoverableProvider
 GenerateIdempotent(context.Context, Request, string, func(string)error)(Result,error)
}
// ModelRecoverer resumes a remote job using its persisted model route.
type ModelIdempotentProvider interface {
 GenerateIdempotent(context.Context,Request,string,func(string)error)(Result,error)
}
type ProviderIdentity interface {
 IdentityFor(string)(string,error)
 ResumeForIdentity(context.Context,string,string,string)(Result,error)
}
type ModelRecoverer interface {
 ResumeFor(context.Context,string,string)(Result,error)
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
// AttachStore recovers known remote jobs without re-submitting paid work.
func (m *Manager) AttachStore(store Store) error {
 if store==nil{return ErrInvalid}
 if _,err:=store.Prune(time.Now().UTC().Add(-7*24*time.Hour));err!=nil{return err}
 existing,err:=store.Load()
 if err!=nil{return err}
 m.mu.Lock()
 defer m.mu.Unlock()
 if len(m.jobs)!=0{return ErrInvalid}
 now:=time.Now().UTC()
 recoverable,canResume:=m.provider.(RecoverableProvider)
 modelRecoverer,canResumeModel:=m.provider.(ModelRecoverer)
 identityRecoverer,hasIdentity:=m.provider.(ProviderIdentity)
 canResume=canResume || canResumeModel
 for _,job:=range existing{
  if job.Status=="queued" || job.Status=="running"{
   if !canResume || job.RemoteID=="" || (hasIdentity && job.ProviderID==""){
    job.Status="failed"
    job.Error="remote job ID unavailable after restart; not resubmitted"
    job.UpdatedAt=now
    if err:=store.Save(job);err!=nil{return err}
   }
  }
  copy:=job
  m.jobs[job.ID]=&copy
 }
 m.store=store
 if canResume {
  for id,job:=range m.jobs {
   if (job.Status=="queued" || job.Status=="running") && job.RemoteID!=""{
    select{
    case m.slots<-struct{}{}:
     ctx,cancel:=context.WithTimeout(context.Background(),m.timeout)
     m.cancel[id]=cancel
     if hasIdentity {
      go m.resumeIdentity(ctx,id,job.Model,job.ProviderID,job.RemoteID,identityRecoverer)
     }else if canResumeModel {
      go m.resumeModel(ctx,id,job.Model,job.RemoteID,modelRecoverer)
     } else {
      go m.resume(ctx,id,job.RemoteID,recoverable)
     }
    default:
     job.Status="failed"
     job.Error="recovery capacity exceeded; remote job not resubmitted"
     job.UpdatedAt=now
     if err:=store.Save(*job);err!=nil{return err}
    }
   }
  }
 }
 return nil
}
func(m *Manager) resumeIdentity(ctx context.Context,id,model,providerID,remoteID string,p ProviderIdentity){
 defer func(){<-m.slots}()
 result,err:=p.ResumeForIdentity(ctx,model,providerID,remoteID)
 m.mu.Lock()
 defer m.mu.Unlock()
 job:=m.jobs[id]
 if job.Status!="cancelled"{
  if err!=nil || ctx.Err()!=nil{job.Status="failed";job.Error="provider identity mismatch or recovery failed"
  }else{job.Status="completed";job.Result=&result}
  job.UpdatedAt=time.Now().UTC()
  if m.store!=nil{
   if saveErr:=m.store.Save(*job);saveErr!=nil{job.Status="failed";job.Error="job storage failure";job.Result=nil}
  }
 }
 m.cancel[id]()
 delete(m.cancel,id)
}
func(m *Manager) resumeModel(ctx context.Context,id,model,remoteID string,p ModelRecoverer){
 defer func(){<-m.slots}()
 result,err:=p.ResumeFor(ctx,model,remoteID)
 m.mu.Lock()
 defer m.mu.Unlock()
 job:=m.jobs[id]
 if job.Status!="cancelled"{
  if err!=nil || ctx.Err()!=nil{job.Status="failed";job.Error="remote recovery failed"
  }else{job.Status="completed";job.Result=&result}
  job.UpdatedAt=time.Now().UTC()
  if m.store!=nil{
   if saveErr:=m.store.Save(*job);saveErr!=nil{job.Status="failed";job.Error="job storage failure";job.Result=nil}
  }
 }
 m.cancel[id]()
 delete(m.cancel,id)
}
func(m *Manager) resume(ctx context.Context,id,remoteID string,p RecoverableProvider){
 defer func(){<-m.slots}()
 result,err:=p.Resume(ctx,remoteID)
 m.mu.Lock()
 defer m.mu.Unlock()
 job:=m.jobs[id]
 if job.Status!="cancelled"{
  if err!=nil || ctx.Err()!=nil{job.Status="failed";job.Error="remote recovery failed"
  }else{job.Status="completed";job.Result=&result}
  job.UpdatedAt=time.Now().UTC()
  if m.store!=nil{
   if saveErr:=m.store.Save(*job);saveErr!=nil{job.Status="failed";job.Error="job storage failure";job.Result=nil}
  }
 }
 m.cancel[id]()
 delete(m.cancel,id)
}
func(m *Manager) recordRemoteID(id,remoteID string)error{
 m.mu.Lock()
 defer m.mu.Unlock()
 job,ok:=m.jobs[id]
 if !ok || job.Status=="cancelled" || remoteID=="" {return ErrInvalid}
 copy:=*job
 copy.RemoteID=remoteID
 copy.UpdatedAt=time.Now().UTC()
 if m.store!=nil {
  if err:=m.store.Save(copy);err!=nil{return ErrStorage}
 }
 *job=copy
 return nil
}
func (m *Manager) Submit(req Request) (Job,error) {
 if req.Model=="" || req.Prompt=="" || len(req.Prompt)>4096 || len(req.Model)>128 {return Job{},ErrInvalid}
 if m.provider==nil {return Job{},ErrUnavailable}
 if catalog,ok:=m.provider.(interface{ProviderFor(string)(Provider,error)});ok {
  if _,err:=catalog.ProviderFor(req.Model);err!=nil{return Job{},ErrUnavailable}
 }
 m.mu.Lock()
 defer m.mu.Unlock()
 if len(m.jobs)>=m.maxJobs {
  // Evict only terminal jobs, oldest first, to avoid blocking new work.
  var oldestID string
  var oldest time.Time
  for id,job:=range m.jobs {
   if job.Status!="completed" && job.Status!="failed" && job.Status!="cancelled" {continue}
   if oldestID=="" || job.UpdatedAt.Before(oldest) {oldestID=id;oldest=job.UpdatedAt}
  }
  if oldestID=="" {return Job{},ErrBusy}
  if m.store!=nil {
   if err:=m.store.Delete(oldestID);err!=nil{return Job{},ErrStorage}
  }
  delete(m.jobs,oldestID)
 }
 select {case m.slots<-struct{}{}:default:return Job{},ErrBusy}
 var bytes [16]byte
 if _,err:=rand.Read(bytes[:]);err!=nil {<-m.slots;return Job{},err}
 id:=hex.EncodeToString(bytes[:])
 now:=time.Now().UTC()
 job:=&Job{ID:id,Status:"queued",Model:req.Model,IdempotencyKey:id,CreatedAt:now,UpdatedAt:now}
 if identity,ok:=m.provider.(ProviderIdentity);ok {
  providerID,err:=identity.IdentityFor(req.Model)
  if err!=nil {<-m.slots;return Job{},ErrUnavailable}
  job.ProviderID=providerID
 }
 ctx,cancel:=context.WithTimeout(context.Background(),m.timeout)
 m.jobs[id]=job
 m.cancel[id]=cancel
 if m.store!=nil {if err:=m.store.Save(*job);err!=nil {delete(m.jobs,id);delete(m.cancel,id);cancel();<-m.slots;return Job{},ErrStorage}}
 go m.run(ctx,id,req)
 return *job,nil
}
func (m *Manager) run(ctx context.Context,id string,req Request) {
 defer func(){<-m.slots}()
 m.mu.Lock()
 if m.jobs[id].Status=="cancelled" {m.mu.Unlock();return}
 m.jobs[id].Status="running"
 m.jobs[id].UpdatedAt=time.Now().UTC()
 if m.store!=nil {
  if err:=m.store.Save(*m.jobs[id]);err!=nil {
   m.jobs[id].Status="failed"
   m.jobs[id].Error="job storage failure"
   m.jobs[id].UpdatedAt=time.Now().UTC()
   m.cancel[id]()
   delete(m.cancel,id)
   m.mu.Unlock()
   return
  }
 }
 m.mu.Unlock()
 var result Result
 var err error
 if p,ok:=m.provider.(ModelIdempotentProvider);ok{
  result,err=p.GenerateIdempotent(ctx,req,id,func(remoteID string)error{return m.recordRemoteID(id,remoteID)})
 }else if p,ok:=m.provider.(IdempotentProvider);ok{
  result,err=p.GenerateIdempotent(ctx,req,id,func(remoteID string)error{return m.recordRemoteID(id,remoteID)})
 }else if p,ok:=m.provider.(RecoverableProvider);ok{
  result,err=p.GenerateTracked(ctx,req,func(remoteID string)error{return m.recordRemoteID(id,remoteID)})
 }else{
  result,err=m.provider.Generate(ctx,req)
 }
 m.mu.Lock()
 defer m.mu.Unlock()
 job:=m.jobs[id]
 if job.Status!="cancelled" {
  if err!=nil {job.Status="failed";job.Error="video generation failed"
  } else if ctx.Err()!=nil {job.Status="failed";job.Error="video generation timed out"
  } else {job.Status="completed";job.Result=&result}
  job.UpdatedAt=time.Now().UTC()
 }
 if m.store!=nil {
  if err:=m.store.Save(*job);err!=nil {
   job.Status="failed"
   job.Error="job storage failure"
   job.Result=nil
  }
 }
 m.cancel[id]()
 delete(m.cancel,id)
}
type ProviderHealth struct{
 Name string `json:"name"`
 Status string `json:"status"`
 LatencyMS int64 `json:"latency_ms"`
 CheckedAt time.Time `json:"checked_at"`
}
func(m *Manager) ProviderHealth(ctx context.Context)[]ProviderHealth{
 if p,ok:=m.provider.(interface{Health(context.Context)[]ProviderHealth});ok{return p.Health(ctx)}
 return []ProviderHealth{}
}
type ModelInfo struct{
 ID string `json:"id"`
 Provider string `json:"provider"`
 Local bool `json:"local"`
}
func(m *Manager) Models()[]ModelInfo{
 if p,ok:=m.provider.(interface{Models() []ModelInfo});ok{return p.Models()}
 return []ModelInfo{}
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
  if m.store!=nil {if err:=m.store.Save(*job);err!=nil{return Job{},ErrStorage}}
  m.cancel[id]()
 }
 return *job,nil
}

func(m *Manager) Prune(before time.Time)(int64,error){
 m.mu.Lock()
 defer m.mu.Unlock()
 if m.store!=nil {
  if _,err:=m.store.Prune(before);err!=nil{return 0,ErrStorage}
 }
 var removed int64
 for id,job:=range m.jobs {
  if (job.Status=="completed" || job.Status=="failed" || job.Status=="cancelled") && job.UpdatedAt.Before(before) {
   delete(m.jobs,id)
   removed++
  }
 }
 return removed,nil
}
