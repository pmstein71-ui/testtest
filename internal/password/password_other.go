//go:build !windows

package password

import (
 "fmt"
 "os"
 "golang.org/x/term"
)

func prompt() (string,error) {
 if !term.IsTerminal(int(os.Stdin.Fd())) {return "",fmt.Errorf("first start requires ADMIN_PASSWORD or an interactive terminal")}
 fmt.Println("Create administrator password. Input is hidden: no characters will be displayed.")
 fmt.Print("Password (12-72 bytes): ")
 b,err:=term.ReadPassword(int(os.Stdin.Fd()));fmt.Println();if err!=nil{return "",err}
 fmt.Print("Confirm password: ")
 confirm,err:=term.ReadPassword(int(os.Stdin.Fd()));fmt.Println();if err!=nil{return "",err}
 if string(b)!=string(confirm){return "",fmt.Errorf("passwords do not match")}
 return string(b),Validate(string(b))
}
