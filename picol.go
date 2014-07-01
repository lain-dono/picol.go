package picol

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	PICOL_OK = iota
	PICOL_ERR
	PICOL_RETURN
	PICOL_BREAK
	PICOL_CONTINUE
)

type Var struct {
	name, val string
	next      *Var
}

type CmdFunc func(i *Interp, argv []string, privdata interface{}) int

type Cmd struct {
	name     string
	fn       CmdFunc
	privdata interface{}
	next     *Cmd
}

type CallFrame struct {
	vars   *Var
	parent *CallFrame
}

type Interp struct {
	level     int
	callframe *CallFrame
	commands  *Cmd
	result    string
}

func InitInterp() *Interp {
	return &Interp{0, &CallFrame{}, nil, ""}
}

func (i *Interp) Result() string {
	return i.result
}
func (i *Interp) SetResult(s string) {
	i.result = s
}

func (i *Interp) GetVar(name string) *Var {
	for v := i.callframe.vars; v != nil; v = v.next {
		if v.name == name {
			return v
		}
	}
	return nil
}

func (i *Interp) SetVar(name, val string) int {
	v := &Var{name, val, i.callframe.vars}
	i.callframe.vars = v
	return PICOL_OK
}

func (i *Interp) GetCommand(name string) *Cmd {
	for c := i.commands; c != nil; c = c.next {
		if c.name == name {
			return c
		}
	}
	return nil
}

func (i *Interp) RegisterCommand(name string, fn CmdFunc, privdata interface{}) int {
	c := i.GetCommand(name)
	if c != nil {
		errbuf := fmt.Sprintf("Command '%s' already defined", name)
		i.SetResult(errbuf)
		return PICOL_ERR
	}

	c = &Cmd{name, fn, privdata, i.commands}
	i.commands = c
	return PICOL_OK
}

/* EVAL! */
func (i *Interp) Eval(t string) int {
	p := InitParser(t)
	i.SetResult("")

	retcode := PICOL_OK

	argc := 0
	argv := []string{}

	for {
		prevtype := p.type_
		// XXX
		_ = p.GetToken()
		if p.type_ == PT_EOF {
			break
		}
		t := p.text[p.start : p.end+1]

		switch p.type_ {
		case PT_VAR:
			//fmt.Printf("PT_VAR token[%d]:'%s'\n", p.type_, t)
			v := i.GetVar(t)
			if v == nil {
				errbuf := fmt.Sprintf("No such variable '%s'", t)
				i.SetResult(errbuf)
				retcode = PICOL_ERR
				goto err
			}
			t = v.val
		case PT_CMD:
			//fmt.Printf("PT_CMD token[%d]:'%s'\n", p.type_, t)
			retcode = i.Eval(t)
			if retcode != PICOL_OK {
				goto err
			}
			t = i.result
		case PT_ESC:
			//fmt.Printf("PT_ESC token[%d]:'%s'\n", p.type_, t)
			// XXX: escape handling missing!
		case PT_SEP:
			//fmt.Printf("PT_SEP token[%d]:'%s'\n", p.type_, t)
			prevtype = p.type_
			continue
		}

		// We have a complete command + args. Call it!
		if p.type_ == PT_EOL {
			prevtype = p.type_
			if argc != 0 {
				c := i.GetCommand(argv[0])
				if c == nil {
					errbuf := fmt.Sprintf("No such command '%s'", argv[0])
					i.SetResult(errbuf)
					retcode = PICOL_ERR
					goto err
				}
				retcode = c.fn(i, argv, c.privdata)
				if retcode != PICOL_OK {
					goto err
				}
			}
			// Prepare for the next command
			argv = []string{}
			argc = 0
			continue
		}

		// We have a new token, append to the previous or as new arg?
		if prevtype == PT_SEP || prevtype == PT_EOL {
			argv = append(argv, t)
			argc++
		} else { // Interpolation
			argv[argc-1] = strings.Join([]string{argv[argc-1], t}, "")
		}
		prevtype = p.type_
	}
err:
	return retcode
}

/* ACTUAL COMMANDS! */
func ArityErr(i *Interp, name string, argv []string) int {
	buf := fmt.Sprintf("Wrong number of args for %s", name, argv)
	i.SetResult(buf)
	return PICOL_ERR
}

func CommandMath(i *Interp, argv []string, pd interface{}) int {
	if len(argv) != 3 {
		return ArityErr(i, argv[0], argv)
	}
	//*
	a, _ := strconv.Atoi(argv[1])
	b, _ := strconv.Atoi(argv[2])
	var c int //*/
	/*
		a, _ := strconv.ParseFloat(argv[1], 64)
		b, _ := strconv.ParseFloat(argv[2], 64)
		var c float64 //*/
	switch {
	case argv[0] == "+":
		c = a + b
	case argv[0] == "-":
		c = a - b
	case argv[0] == "*":
		c = a * b
	case argv[0] == "/":
		c = a / b
	case argv[0] == ">":
		if a > b {
			c = 1
		}
	case argv[0] == ">=":
		if a >= b {
			c = 1
		}
	case argv[0] == "<":
		if a < b {
			c = 1
		}
	case argv[0] == "<=":
		if a <= b {
			c = 1
		}
	case argv[0] == "==":
		if a == b {
			c = 1
		}
	case argv[0] == "!=":
		if a != b {
			c = 1
		}
	default: // FIXME I hate warnings
		c = 0
	}
	buf := fmt.Sprintf("%d", c)
	i.SetResult(buf)
	return PICOL_OK
}

func CommandSet(i *Interp, argv []string, pd interface{}) int {
	if len(argv) != 3 {
		return ArityErr(i, argv[0], argv)
	}
	i.SetVar(argv[1], argv[2])
	i.SetResult(argv[2])
	return PICOL_OK
}

func CommandPuts(i *Interp, argv []string, pd interface{}) int {
	if len(argv) != 2 {
		return ArityErr(i, argv[0], argv)
	}
	fmt.Println(argv[1])
	return PICOL_OK
}

func CommandIf(i *Interp, argv []string, pd interface{}) int {
	if len(argv) != 3 && len(argv) != 5 {
		return ArityErr(i, argv[0], argv)
	}
	if retcode := i.Eval(argv[1]); retcode != PICOL_OK {
		return retcode
	}
	if r, _ := strconv.Atoi(i.result); r != 0 {
		return i.Eval(argv[2])
	} else if len(argv) == 5 {
		return i.Eval(argv[4])
	}
	return PICOL_OK
}

func CommandWhile(i *Interp, argv []string, pd interface{}) int {
	if len(argv) != 3 {
		return ArityErr(i, argv[0], argv)
	}
	for {
		retcode := i.Eval(argv[1])
		if retcode != PICOL_OK {
			return retcode
		}
		if r, _ := strconv.Atoi(i.result); r != 0 {
			retcode = i.Eval(argv[2])
			switch retcode {
			case PICOL_CONTINUE, PICOL_OK:
				//pass
			case PICOL_BREAK:
				return PICOL_OK
			default:
				return retcode
			}
		} else {
			return PICOL_OK
		}
	}
}

func CommandRetCodes(i *Interp, argv []string, pd interface{}) int {
	if len(argv) != 1 {
		return ArityErr(i, argv[0], argv)
	}
	switch argv[0] {
	case "break":
		return PICOL_BREAK
	case "continue":
		return PICOL_CONTINUE
	}
	return PICOL_OK
}

func DropCallFrame(i *Interp) {
	i.callframe = i.callframe.parent
}

func CommandCallProc(i *Interp, argv []string, pd interface{}) int {
	var x []string

	if pd, ok := pd.([]string); ok {
		x = pd
	} else {
		return PICOL_OK
	}

	alist := x[0]
	body := x[1]
	p := alist[:]
	arity := 0

	done := false

	errcode := PICOL_OK

	cf := &CallFrame{vars: nil, parent: i.callframe}
	i.callframe = cf

	for {
		start := p
		for len(p) != 0 && p[0] != ' ' {
			p = p[1:]
		}
		if len(p) != 0 && p == start {
			p = p[1:]
			continue
		}

		if p == start {
			break
		}
		if len(p) == 0 {
			done = true
		} else {
			p = p[1:1]
		}
		arity++
		if arity > len(argv)-1 {
			goto arityerr
		}
		i.SetVar(start, argv[arity])
		if len(p) != 0 {
			p = p[1:]
		}
		if done {
			break
		}
	}

	if arity != len(argv)-1 {
		goto arityerr
	}
	errcode = i.Eval(body)
	//fmt.Println("eval", errcode)
	if errcode == PICOL_RETURN {
		errcode = PICOL_OK
	}
	DropCallFrame(i) // remove the called proc callframe
	return errcode
arityerr:
	errbuf := fmt.Sprintf("Proc '%s' called with wrong arg num", argv[0])
	i.SetResult(errbuf)
	DropCallFrame(i) // remove the called proc callframe
	return PICOL_ERR
}

func CommandProc(i *Interp, argv []string, pd interface{}) int {
	if len(argv) != 4 {
		return ArityErr(i, argv[0], argv)
	}
	// FIXME maybe create copy
	procdata := []string{argv[2], argv[3]}
	return i.RegisterCommand(argv[1], CommandCallProc, procdata)
}

func CommandReturn(i *Interp, argv []string, pd interface{}) int {
	if len(argv) != 1 && len(argv) != 2 {
		return ArityErr(i, argv[0], argv)
	}
	var r string
	if len(argv) == 2 {
		r = argv[1]
	}
	i.SetResult(r)
	return PICOL_RETURN
}

func (i *Interp) RegisterCoreCommands() {
	name := [...]string{"+", "-", "*", "/", ">", ">=", "<", "<=", "==", "!="}
	for _, n := range name {
		i.RegisterCommand(n, CommandMath, nil)
	}
	i.RegisterCommand("set", CommandSet, nil)
	i.RegisterCommand("puts", CommandPuts, nil)
	i.RegisterCommand("if", CommandIf, nil)
	i.RegisterCommand("while", CommandWhile, nil)
	i.RegisterCommand("break", CommandRetCodes, nil)
	i.RegisterCommand("continue", CommandRetCodes, nil)
	i.RegisterCommand("proc", CommandProc, nil)
	i.RegisterCommand("return", CommandReturn, nil)
}
