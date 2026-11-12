package jobs

import (
 "database/sql"
 "encoding/json"
 "errors"
 "fmt"
 "time"

 _ "github.com/mattn/go-sqlite3"
)

type Store interface {
 Save(Job) error
 Load() ([]Job,error)
 Prune(time.Time) (int64,error)
 Delete(string) error
 Close() error
}

type SQLiteStore struct{db *sql.DB}

func OpenSQLite(path string)(*SQLiteStore,error){
 if path=="" {return nil,errors.New("database path required")}
 db,err:=sql.Open("sqlite3",path)
 if err!=nil{return nil,err}
 db.SetMaxOpenConns(1)
 if _,err=db.Exec(`PRAGMA busy_timeout=5000; PRAGMA journal_mode=WAL; CREATE TABLE IF NOT EXISTS video_jobs (id TEXT PRIMARY KEY, updated_at TEXT NOT NULL, payload BLOB NOT NULL);`);err!=nil{db.Close();return nil,fmt.Errorf("initialize job database: %w",err)}
 return &SQLiteStore{db:db},nil
}
func(s *SQLiteStore)Save(job Job)error{
 payload,err:=json.Marshal(job)
 if err!=nil{return err}
 _,err=s.db.Exec("INSERT INTO video_jobs(id,updated_at,payload) VALUES(?,?,?) ON CONFLICT(id) DO UPDATE SET updated_at=excluded.updated_at,payload=excluded.payload",job.ID,job.UpdatedAt.UTC().Format(time.RFC3339Nano),payload)
 return err
}
func(s *SQLiteStore)Load()([]Job,error){
 rows,err:=s.db.Query("SELECT payload FROM video_jobs ORDER BY updated_at DESC")
 if err!=nil{return nil,err}
 defer rows.Close()
 var jobs []Job
 for rows.Next(){
  var raw []byte
  if err=rows.Scan(&raw);err!=nil{return nil,err}
  var job Job
  if err=json.Unmarshal(raw,&job);err!=nil{return nil,err}
  jobs=append(jobs,job)
 }
 return jobs,rows.Err()
}
func(s *SQLiteStore)Prune(before time.Time)(int64,error){
 res,err:=s.db.Exec("DELETE FROM video_jobs WHERE updated_at < ? AND json_extract(payload,'$.status') IN ('completed','failed','cancelled')",before.UTC().Format(time.RFC3339Nano))
 if err!=nil{return 0,err}
 return res.RowsAffected()
}
func(s *SQLiteStore)Close()error{return s.db.Close()}

func(s *SQLiteStore)Delete(id string)error{
 _,err:=s.db.Exec("DELETE FROM video_jobs WHERE id = ?",id)
 return err
}
