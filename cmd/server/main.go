package main

import (
 "context"
 "crypto/rand"
 "crypto/sha256"
 "database/sql"
 "encoding/hex"
 "encoding/json"
 "fmt"
 "log"
 "net/http"
 "os"
 "os/signal"
 "path/filepath"
 "strings"
 "sync"
 "syscall"
 "time"

 "golang.org/x/crypto/bcrypt"
 "github.com/pmstein71-ui/testtest/internal/password"
 "github.com/pmstein71-ui/testtest/internal/startup"
 _ "modernc.org/sqlite"
)

func env(k,d string) string { if v:=os.Getenv(k);v!="" {return v};return d }
func main(){startup.Run("server",run)}
func run() error {
 cert,key:=os.Getenv("TLS_CERT"),os.Getenv("TLS_KEY");if (cert=="")!=(key==""){return fmt.Errorf("TLS_CERT and TLS_KEY must be set together")}
 dir,err:=startup.DataDir();if err!=nil{return err};log.Printf("Database directory: %s",dir); if err:=os.MkdirAll(dir,0700);err!=nil{return err}
 db,err:=sql.Open("sqlite",filepath.Join(dir,"app.db"));if err!=nil{return err};defer db.Close();db.SetMaxOpenConns(1)
 for _,q:=range []string{`PRAGMA journal_mode=WAL`, `PRAGMA busy_timeout=5000`, `CREATE TABLE IF NOT EXISTS users (id INTEGER PRIMARY KEY, login TEXT UNIQUE NOT NULL, password BLOB NOT NULL)`, `CREATE TABLE IF NOT EXISTS sessions (token TEXT PRIMARY KEY, user_id INTEGER NOT NULL, expires INTEGER NOT NULL)`}{if _,err=db.Exec(q);err!=nil{return err}}
 var count int;if err=db.QueryRow(`SELECT count(*) FROM users`).Scan(&count);err!=nil{return err}
 if count==0 {log.Println("First start: create administrator password in the setup window (Windows), or set ADMIN_PASSWORD")
 p,e:=password.Read();if e!=nil{return e};hash,e:=bcrypt.GenerateFromPassword([]byte(p),bcrypt.DefaultCost);if e!=nil{return e};if _,err=db.Exec(`INSERT INTO users(login,password) VALUES(?,?)`,env("ADMIN_LOGIN","admin"),hash);err!=nil{return err}}
 interval,err:=time.ParseDuration(env("BACKUP_INTERVAL","24h"));if err!=nil||interval<time.Minute{return fmt.Errorf("BACKUP_INTERVAL must be a duration >= 1m")}
 backupDir:=env("BACKUP_DIR",filepath.Join(dir,"backups"));if err=os.MkdirAll(backupDir,0700);err!=nil{return err}
 backup:=func(){name:=filepath.Join(backupDir,"backup-"+time.Now().UTC().Format("20060102T150405.000000000Z")+".db");_,e:=db.Exec(`VACUUM INTO '`+strings.ReplaceAll(name,"'","''")+`'`);if e!=nil{log.Printf("Backup failed: %v",e)}else{log.Printf("Backup saved: %s",name)}}
 ctx,stop:=signal.NotifyContext(context.Background(),os.Interrupt,syscall.SIGTERM);defer stop()
 go func(){backup();t:=time.NewTicker(interval);defer t.Stop();for{select{case <-ctx.Done():return;case <-t.C:backup();_,e:=db.Exec(`DELETE FROM sessions WHERE expires < ?`,time.Now().Unix());if e!=nil{log.Print(e)}}}}()
 mux:=http.NewServeMux();var loginMu sync.Mutex;var nextLogin time.Time
 mux.HandleFunc("POST /api/login",func(w http.ResponseWriter,r *http.Request){
 // Global throttle limits brute force and avoids unbounded per-address state.
 loginMu.Lock();if time.Now().Before(nextLogin){loginMu.Unlock();http.Error(w,"Try again shortly",429);return};nextLogin=time.Now().Add(time.Second);loginMu.Unlock()
 var in struct{Login string `json:"login"`;Password string `json:"password"`};if json.NewDecoder(http.MaxBytesReader(w,r.Body,4096)).Decode(&in)!=nil{http.Error(w,"Invalid request",400);return}
 var id int;var hash []byte;err:=db.QueryRow(`SELECT id,password FROM users WHERE login=?`,in.Login).Scan(&id,&hash)
 if err!=nil||bcrypt.CompareHashAndPassword(hash,[]byte(in.Password))!=nil{http.Error(w,"Invalid credentials",401);return}
 b:=make([]byte,32);if _,err=rand.Read(b);err!=nil{http.Error(w,"Random source unavailable",500);return};token:=hex.EncodeToString(b);sum:=sha256.Sum256([]byte(token))
 if _,err=db.Exec(`INSERT INTO sessions(token,user_id,expires) VALUES(?,?,?)`,hex.EncodeToString(sum[:]),id,time.Now().Add(8*time.Hour).Unix());err!=nil{http.Error(w,"Database error",500);return};w.Header().Set("Content-Type","application/json");json.NewEncoder(w).Encode(map[string]string{"token":token})
 })
 mux.HandleFunc("GET /api/me",func(w http.ResponseWriter,r *http.Request){sum:=sha256.Sum256([]byte(strings.TrimPrefix(r.Header.Get("Authorization"),"Bearer ")));var login string;err:=db.QueryRow(`SELECT u.login FROM sessions s JOIN users u ON u.id=s.user_id WHERE s.token=? AND s.expires>?`,hex.EncodeToString(sum[:]),time.Now().Unix()).Scan(&login);if err!=nil{http.Error(w,"Unauthorized",401);return};w.Header().Set("Content-Type","application/json");json.NewEncoder(w).Encode(map[string]string{"login":login,"message":"Подключение к серверу и базе данных работает"})})
 mux.HandleFunc("POST /api/logout",func(w http.ResponseWriter,r *http.Request){sum:=sha256.Sum256([]byte(strings.TrimPrefix(r.Header.Get("Authorization"),"Bearer ")));if _,err:=db.Exec(`DELETE FROM sessions WHERE token=?`,hex.EncodeToString(sum[:]));err!=nil{http.Error(w,"Database error",500);return};w.WriteHeader(204)})
 srv:=&http.Server{Addr:env("LISTEN_ADDR","127.0.0.1:8080"),Handler:http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){w.Header().Set("Cache-Control","no-store");mux.ServeHTTP(w,r)}),ReadHeaderTimeout:5*time.Second,ReadTimeout:15*time.Second,WriteTimeout:30*time.Second,IdleTimeout:60*time.Second}
 go func(){<-ctx.Done();c,cancel:=context.WithTimeout(context.Background(),10*time.Second);defer cancel();srv.Shutdown(c)}()
 log.Println("Server listening on",srv.Addr)
 if cert!=""&&key!=""{err=srv.ListenAndServeTLS(cert,key)}else{log.Print("HTTP mode: use only on loopback or behind an HTTPS reverse proxy");err=srv.ListenAndServe()};if err!=nil&&err!=http.ErrServerClosed{return err};return nil
}
