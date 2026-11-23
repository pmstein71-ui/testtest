// Package startup keeps failures visible when Windows executables are double-clicked.
package startup

import (
 "fmt"
 "io"
 "log"
 "os"
 "os/exec"
 "path/filepath"
 "runtime"
 "runtime/debug"
)

func Run(name string, run func() error) {
 var file *os.File
 root, err := os.UserConfigDir()
 if err == nil {
  dir := filepath.Join(root,"Pulse","logs")
  if os.MkdirAll(dir,0700)==nil {file,err=os.OpenFile(filepath.Join(dir,name+".log"),os.O_CREATE|os.O_APPEND|os.O_WRONLY,0600)}
 }
 if file!=nil {defer file.Close();log.SetOutput(io.MultiWriter(os.Stderr,file))}
 log.Printf("Starting %s",name)
 failure:=func(err error){
  log.Printf("%s stopped: %v",name,err)
  if runtime.GOOS=="windows"&&os.Getenv("PULSE_NO_DIALOG")!="1" {
   message:=fmt.Sprintf("Pulse %s could not start.\n%s\nLogs: %s\\Pulse\\logs\\%s.log",name,err,root,name)
   cmd:=exec.Command("powershell.exe","-NoProfile","-NonInteractive","-Command",`Add-Type -AssemblyName System.Windows.Forms; [void][System.Windows.Forms.MessageBox]::Show($env:PULSE_ERROR_MESSAGE, 'Pulse - startup error')`)
   cmd.Env=append(os.Environ(),"PULSE_ERROR_MESSAGE="+message)
   if e:=cmd.Run();e!=nil{log.Printf("Cannot display error dialog: %v",e)}
  }
 }
 defer func(){if r:=recover();r!=nil{log.Printf("panic: %s",debug.Stack());failure(fmt.Errorf("%v",r));os.Exit(1)}}()
 if err:=run();err!=nil{failure(err);if file!=nil{file.Close()};os.Exit(1)}
}

// DataDir preserves an existing legacy database, otherwise uses a stable per-user path.
func DataDir() (string,error) {
 if dir:=os.Getenv("DATA_DIR");dir!="" {return dir,nil}
 if _,err:=os.Stat(filepath.Join("data","app.db"));err==nil{return "data",nil}
 root,err:=os.UserConfigDir();if err!=nil{return "",err}
 return filepath.Join(root,"PulseServer","data"),nil
}
