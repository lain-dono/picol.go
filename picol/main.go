package main

import (
	"../../picol.go"
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
)

var fname = flag.String("f", "", "file name")

func CommandPuts(i *picol.Interp, argv []string, pd interface{}) int {
	if len(argv) != 2 {
		i.Result = fmt.Sprintf("Wrong number of args for %s %s", argv[0], argv)
		return picol.PICOL_ERR
	}
	fmt.Println(argv[1])
	return picol.PICOL_OK
}

func main() {
	flag.Parse()
	interp := picol.InitInterp()
	interp.RegisterCoreCommands()
	interp.RegisterCommand("puts", CommandPuts, nil)

	buf, err := ioutil.ReadFile(*fname)
	if err == nil {
		retcode := interp.Eval(string(buf))
		if retcode != picol.PICOL_OK {
			fmt.Printf("[%d] %s\n", retcode, interp.Result)
		}
	} else {
		for {
			fmt.Print("picol> ")
			scanner := bufio.NewReader(os.Stdin)
			clibuf, _ := scanner.ReadString('\n')
			retcode := interp.Eval(clibuf[:len(clibuf)-1])
			if len(interp.Result) != 0 {
				fmt.Printf("[%d] %s\n", retcode, interp.Result)
			}
		}
	}
}
