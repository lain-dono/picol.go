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

type Var string
type CmdFunc func(i *Interp, argv []string, privdata interface{}) int
type Cmd struct {
	fn       CmdFunc
	privdata interface{}
}
type CallFrame struct {
	vars   map[string]Var
	parent *CallFrame
}
type Interp struct {
	level     int
	callframe *CallFrame
	commands  map[string]Cmd
	Result    string
}

func InitInterp() *Interp {
	return &Interp{
		level:     0,
		callframe: &CallFrame{vars: make(map[string]Var)},
		commands:  make(map[string]Cmd),
		Result:    ""}
}

func (i *Interp) Var(name string) (Var, bool) {
	v, ok := i.callframe.vars[name]
	return v, ok
}

func (i *Interp) SetVar(name, val string) {
	i.callframe.vars[name] = Var(val)
}

func (i *Interp) Command(name string) *Cmd {
	v, ok := i.commands[name]
	if !ok {
		return nil
	}
	return &v
}

func (i *Interp) RegisterCommand(name string, fn CmdFunc, privdata interface{}) int {
	c := i.Command(name)
	if c != nil {
		i.Result = fmt.Sprintf("Command '%s' already defined", name)
		return PICOL_ERR
	}

	i.commands[name] = Cmd{fn, privdata}
	return PICOL_OK
}

/* EVAL! */
func (i *Interp) Eval(t string) int {
	p := InitParser(t)
	i.Result = ""

	argv := []string{}

	for {
		prevtype := p.type_
		// XXX
		t = p.GetToken()
		if p.type_ == PT_EOF {
			break
		}

		switch p.type_ {
		case PT_VAR:
			v, ok := i.Var(t)
			if !ok {
				i.Result = fmt.Sprintf("No such variable '%s'", t)
				return PICOL_ERR
			}
			t = string(v)
		case PT_CMD:
			if code := i.Eval(t); code != PICOL_OK {
				return code
			}
			t = i.Result
		case PT_ESC:
			// XXX: escape handling missing!
		case PT_SEP:
			prevtype = p.type_
			continue
		}

		// We have a complete command + args. Call it!
		if p.type_ == PT_EOL {
			prevtype = p.type_
			if len(argv) != 0 {
				c := i.Command(argv[0])
				if c == nil {
					i.Result = fmt.Sprintf("No such command '%s'", argv[0])
					return PICOL_ERR
				}
				if code := c.fn(i, argv, c.privdata); code != PICOL_OK {
					return code
				}
			}
			// Prepare for the next command
			argv = []string{}
			continue
		}

		// We have a new token, append to the previous or as new arg?
		if prevtype == PT_SEP || prevtype == PT_EOL {
			argv = append(argv, t)
		} else { // Interpolation
			argv[len(argv)-1] = strings.Join([]string{argv[len(argv)-1], t}, "")
		}
		prevtype = p.type_
	}
	return PICOL_OK
}

/* ACTUAL COMMANDS! */
func arityErr(i *Interp, name string, argv []string) int {
	i.Result = fmt.Sprintf("Wrong number of args for %s %s", name, argv)
	return PICOL_ERR
}

func CommandMath(i *Interp, argv []string, pd interface{}) int {
	if len(argv) != 3 {
		return arityErr(i, argv[0], argv)
	}
	a, _ := strconv.Atoi(argv[1])
	b, _ := strconv.Atoi(argv[2])
	var c int
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
	i.Result = fmt.Sprintf("%d", c)
	return PICOL_OK
}

func CommandSet(i *Interp, argv []string, pd interface{}) int {
	if len(argv) != 3 {
		return arityErr(i, argv[0], argv)
	}
	i.SetVar(argv[1], argv[2])
	i.Result = argv[2]
	return PICOL_OK
}

func CommandIf(i *Interp, argv []string, pd interface{}) int {
	if len(argv) != 3 && len(argv) != 5 {
		return arityErr(i, argv[0], argv)
	}
	if retcode := i.Eval(argv[1]); retcode != PICOL_OK {
		return retcode
	}
	if r, _ := strconv.Atoi(i.Result); r != 0 {
		return i.Eval(argv[2])
	} else if len(argv) == 5 {
		return i.Eval(argv[4])
	}
	return PICOL_OK
}

func CommandWhile(i *Interp, argv []string, pd interface{}) int {
	if len(argv) != 3 {
		return arityErr(i, argv[0], argv)
	}
	for {
		retcode := i.Eval(argv[1])
		if retcode != PICOL_OK {
			return retcode
		}
		if r, _ := strconv.Atoi(i.Result); r != 0 {
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
		return arityErr(i, argv[0], argv)
	}
	switch argv[0] {
	case "break":
		return PICOL_BREAK
	case "continue":
		return PICOL_CONTINUE
	}
	return PICOL_OK
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
	i.callframe = &CallFrame{vars: make(map[string]Var), parent: i.callframe}
	defer func() { i.callframe = i.callframe.parent }() // remove the called proc callframe

	err := 0

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
			i.Result = fmt.Sprintf("Proc '%s' called with wrong arg num", argv[0])
			return PICOL_ERR
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
		i.Result = fmt.Sprintf("Proc '%s' called with wrong arg num", argv[0])
		return PICOL_ERR
	}
	err = i.Eval(body)
	if err == PICOL_RETURN {
		err = PICOL_OK
	}
	return err
}

func CommandProc(i *Interp, argv []string, pd interface{}) int {
	if len(argv) != 4 {
		return arityErr(i, argv[0], argv)
	}
	return i.RegisterCommand(argv[1], CommandCallProc, []string{argv[2], argv[3]})
}

func CommandReturn(i *Interp, argv []string, pd interface{}) int {
	if len(argv) != 1 && len(argv) != 2 {
		return arityErr(i, argv[0], argv)
	}
	var r string
	if len(argv) == 2 {
		r = argv[1]
	}
	i.Result = r
	return PICOL_RETURN
}

func (i *Interp) RegisterCoreCommands() {
	name := [...]string{"+", "-", "*", "/", ">", ">=", "<", "<=", "==", "!="}
	for _, n := range name {
		i.RegisterCommand(n, CommandMath, nil)
	}
	i.RegisterCommand("set", CommandSet, nil)
	i.RegisterCommand("if", CommandIf, nil)
	i.RegisterCommand("while", CommandWhile, nil)
	i.RegisterCommand("break", CommandRetCodes, nil)
	i.RegisterCommand("continue", CommandRetCodes, nil)
	i.RegisterCommand("proc", CommandProc, nil)
	i.RegisterCommand("return", CommandReturn, nil)
}
