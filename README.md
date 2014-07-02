# picol.go

Original http://oldblog.antirez.com/post/picol.html

Sample use:
```golang
func CommandPuts(i *picol.Interp, argv []string, pd interface{}) int {
	if len(argv) != 2 {
		i.Result = fmt.Sprintf("Wrong number of args for %s %s", argv[0], argv)
		return picol.PICOL_ERR
	}
	fmt.Println(argv[1])
	return picol.PICOL_OK
}
...
	interp := picol.InitInterp()
	// add core functions
	interp.RegisterCoreCommands()
	// add user function
	interp.RegisterCommand("puts", CommandPuts, nil)
	// eval
	retcode := interp.Eval(string(buf))
	if retcode != picol.PICOL_OK {
		fmt.Printf("[%d] %s\n", retcode, interp.Result)
	}
```
