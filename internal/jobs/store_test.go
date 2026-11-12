package jobs

import (
 "path/filepath"
 "testing"
 "time"
)
func TestSQLitePersistenceAndPrune(t *testing.T){
 path:=filepath.Join(t.TempDir(),"jobs.sqlite")
 store,err:=OpenSQLite(path)
 if err!=nil{t.Fatal(err)}
 old:=time.Now().UTC().Add(-48*time.Hour)
 active:=Job{ID:"running",Status:"running",CreatedAt:old,UpdatedAt:old}
 done:=Job{ID:"done",Status:"completed",CreatedAt:old,UpdatedAt:old,Result:&Result{URL:"https://example.invalid/video.mp4",MIMEType:"video/mp4"}}
 if err=store.Save(active);err!=nil{t.Fatal(err)}
 if err=store.Save(done);err!=nil{t.Fatal(err)}
 if err=store.Close();err!=nil{t.Fatal(err)}
 store,err=OpenSQLite(path)
 if err!=nil{t.Fatal(err)}
 defer store.Close()
 loaded,err:=store.Load()
 if err!=nil || len(loaded)!=2{t.Fatalf("loaded %d: %v",len(loaded),err)}
 n,err:=store.Prune(time.Now().UTC().Add(-24*time.Hour))
 if err!=nil || n!=1{t.Fatalf("pruned %d: %v",n,err)}
 loaded,err=store.Load()
 if err!=nil || len(loaded)!=1 || loaded[0].ID!="running"{t.Fatalf("remaining %+v: %v",loaded,err)}
}
