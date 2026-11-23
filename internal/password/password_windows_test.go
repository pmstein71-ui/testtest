package password

import (
 "encoding/base64"
 "encoding/binary"
 "os/exec"
 "strings"
 "testing"
 "unicode/utf16"
)
func TestEncodedCommand(t *testing.T){
 b,err:=base64.StdEncoding.DecodeString(encodedCommand(dialogScript));if err!=nil{t.Fatal(err)}
 units:=make([]uint16,len(b)/2);for i:=range units{units[i]=binary.LittleEndian.Uint16(b[i*2:])}
 if string(utf16.Decode(units))!=dialogScript{t.Fatal("script Unicode corrupted")}
}
func TestPasswordTransport(t *testing.T){
 expected:="пароль-123456789"
 got,err:=decodePassword([]byte(base64.StdEncoding.EncodeToString([]byte(expected))+"\r\n"));if err!=nil||got!=expected{t.Fatal("password transport corrupted")}
 for _,bad:=range []string{"not base64!","",base64.StdEncoding.EncodeToString([]byte("short"))}{if _,err:=decodePassword([]byte(bad));err==nil{t.Fatal("invalid output accepted")}}
}
func TestDialogScriptParses(t *testing.T){
 // Parse the actual embedded script without opening a window on the CI runner.
 script:=`$errors = $null; $tokens = $null; [void][System.Management.Automation.Language.Parser]::ParseInput([Text.Encoding]::Unicode.GetString([Convert]::FromBase64String('`+encodedCommand(dialogScript)+`')), [ref]$tokens, [ref]$errors); if ($errors.Count) { exit 1 }`
 cmd:=exec.Command("powershell.exe","-NoProfile","-NonInteractive","-EncodedCommand",encodedCommand(script))
 if err:=cmd.Run();err!=nil{t.Fatal("PowerShell dialog script syntax invalid")}
 for _,required:=range []string{"UseSystemPasswordChar = $true","ShortcutsEnabled = $true","-cne","GetByteCount","$form.AcceptButton","$form.CancelButton"}{if !strings.Contains(dialogScript,required){t.Fatalf("dialog missing %s",required)}}
}
