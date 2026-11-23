package password

import (
 _ "embed"
 "encoding/base64"
 "encoding/binary"
 "fmt"
 "os/exec"
 "strings"
 "syscall"
 "unicode/utf16"
)

//go:embed prompt.ps1
var dialogScript string

func encodedCommand(script string) string {
 units:=utf16.Encode([]rune(script));b:=make([]byte,len(units)*2)
 for i,v:=range units {binary.LittleEndian.PutUint16(b[i*2:],v)}
 return base64.StdEncoding.EncodeToString(b)
}

func decodePassword(output []byte) (string,error) {
 b,err:=base64.StdEncoding.DecodeString(strings.TrimSpace(string(output)))
 if err!=nil {return "",fmt.Errorf("invalid password dialog response")}
 value:=string(b)
 return value,Validate(value)
}

func prompt() (string,error) {
 cmd:=exec.Command("powershell.exe","-NoLogo","-NoProfile","-NonInteractive","-STA","-EncodedCommand",encodedCommand(dialogScript))
 // No terminal input is used; Windows Forms handles keyboard, paste and masking.
 cmd.SysProcAttr=&syscall.SysProcAttr{HideWindow:true}
 out,err:=cmd.Output()
 if err!=nil {
  if e,ok:=err.(*exec.ExitError);ok&&e.ExitCode()==2 {return "",fmt.Errorf("administrator setup cancelled; start Pulse Server again to retry")}
  // Never include command output: it may contain the password.
  return "",fmt.Errorf("cannot open administrator setup window; use ADMIN_PASSWORD in PowerShell as described in README.md")
 }
 return decodePassword(out)
}
