package startup

import (
 "os"
 "path/filepath"
 "testing"
)
func TestExplicitDataDir(t *testing.T){t.Setenv("DATA_DIR",t.TempDir());got,err:=DataDir();if err!=nil||got!=os.Getenv("DATA_DIR"){t.Fatalf("%s %v",got,err)}}
func TestStableDataDir(t *testing.T){
 t.Setenv("DATA_DIR","");cwd,err:=os.Getwd();if err!=nil{t.Fatal(err)};defer os.Chdir(cwd)
 if err=os.Chdir(t.TempDir());err!=nil{t.Fatal(err)}
 root,err:=os.UserConfigDir();if err!=nil{t.Fatal(err)};got,err:=DataDir();if err!=nil||got!=filepath.Join(root,"PulseServer","data"){t.Fatalf("%s %v",got,err)}
 if err=os.Mkdir("data",0700);err!=nil{t.Fatal(err)};if err=os.WriteFile(filepath.Join("data","app.db"),nil,0600);err!=nil{t.Fatal(err)}
 got,err=DataDir();if err!=nil||got!="data"{t.Fatalf("legacy database not preserved: %s %v",got,err)}
}
