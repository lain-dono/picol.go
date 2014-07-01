package main

//import "fmt"

const (
	PT_ESC = iota
	PT_STR
	PT_CMD
	PT_VAR
	PT_SEP
	PT_EOL
	PT_EOF
)

type Parser struct {
	text              string
	p, start, end, ln int
	insidequote       int
	type_             int
}

func InitParser(text string) *Parser {
	return &Parser{text, 0, 0, 0, len(text), 0, PT_EOL}
}

func (p *Parser) ParseSep() int {
	p.start = p.p
	c := p.text[p.p]
	for ; c == ' ' || c == '\t' || c == '\n' || c == '\r'; c = p.text[p.p] {
		p.p++
		p.ln--
		if p.p == len(p.text) {
			break
		}
	}
	p.end = p.p - 1
	p.type_ = PT_SEP
	return PICOL_OK
}

func (p *Parser) ParseEol() int {
	// XXX add ';' and PT_EOL
	p.start = p.p

	c := p.text[p.p]
	for ; c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == ';'; c = p.text[p.p] {
		p.p++
		p.ln--
		if p.p == len(p.text) {
			break
		}
	}
	p.end = p.p - 1
	p.type_ = PT_EOL
	return PICOL_OK
}

func (p *Parser) ParseCommand() int {
	level, blevel := 1, 0

	p.p++
	p.ln--
	p.start = p.p

Loop:
	for {
		//fmt.Println("lvl", level)
		switch {
		//case p.ln == 0 || len(p.text)-1 >= p.p:
		case p.ln == 0:
			//fmt.Println("  break", p.ln, p.p)
			break Loop
		case p.text[p.p] == '[' && blevel == 0:
			//fmt.Println("  lvl++")
			level++
		case p.text[p.p] == ']' && blevel == 0:
			//fmt.Println("  lvl--")
			level--
			if level == 0 {
				break Loop
			}
		case p.text[p.p] == '\\':
			//fmt.Println("  \\\\\\")
			p.p++
			p.ln--

		case p.text[p.p] == '{':
			//fmt.Println("  {{{{{")
			blevel++
		case p.text[p.p] == '}':
			//fmt.Println("  }}}}")
			if blevel != 0 {
				blevel--
			}
		}
		p.p++
		p.ln--
	}
	//fmt.Println("end")
	p.end = p.p - 1
	p.type_ = PT_CMD

	if p.text[p.p] == ']' {
		p.p++
		p.ln--
	}
	return PICOL_OK
}

func (p *Parser) ParseVar() int {
	// skip the $
	p.p++
	p.ln--
	p.start = p.p

	for {
		if p.p == len(p.text) {
			break
		}
		c := p.text[p.p]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' {
			p.p++
			p.ln--
			continue
		}
		break
	}

	if p.start == p.p { // It's just a single char string "$"
		p.start = p.p - 1
		p.end = p.p - 1
		p.type_ = PT_STR
	} else {
		p.end = p.p - 1
		p.type_ = PT_VAR
	}
	return PICOL_OK
}

func (p *Parser) ParseBrace() int {
	level := 1
	p.p++
	p.ln--
	p.start = p.p

	for {
		c := p.text[p.p]
		switch {
		case p.ln >= 2 && c == '\\':
			p.p++
			p.ln--
		case p.ln == 0 || c == '}':
			level--
			if level == 0 || p.ln == 0 {
				p.end = p.p - 1
				if p.ln != 0 {
					// Skip final closed brace
					p.p++
					p.ln--

				}
				p.type_ = PT_STR
				return PICOL_OK
			}
		case c == '{':
			level++
		}
		p.p++
		p.ln--
	}
	return PICOL_OK /* unreached */
}

func (p *Parser) ParseString() int {
	newword := p.type_ == PT_SEP || p.type_ == PT_EOL || p.type_ == PT_STR
	c := p.text[p.p]
	if newword && c == '{' {
		return p.ParseBrace()
	} else if newword && c == '"' {
		p.insidequote = 1
		p.p++
		p.ln--
	}

	p.start = p.p

	for {
		if p.ln == 0 {
			p.end = p.p - 1
			p.type_ = PT_ESC
			return PICOL_OK
		}
		switch p.text[p.p] {
		case '\\':
			if p.ln >= 2 {
				p.p++
				p.ln--
			}
		case '$':
		case '[':
			p.end = p.p - 1
			p.type_ = PT_ESC
			return PICOL_OK
		case ' ', '\t', '\n', '\r', ';':
			if p.insidequote == 0 {
				p.end = p.p - 1
				p.type_ = PT_ESC
				return PICOL_OK
			}
		case '"':
			if p.insidequote != 0 {
				p.end = p.p - 1
				p.type_ = PT_ESC
				p.p++
				p.ln--
				p.insidequote = 0
				return PICOL_OK
			}
		}
		p.p++
		p.ln--
	}
	return PICOL_OK /* unreached */
}

func (p *Parser) ParseComment() int {
	for p.ln != 0 && p.text[p.p] != '\n' {
		p.p++
		p.ln--
	}
	return PICOL_OK
}

func (p *Parser) GetToken() int {
	for {
		if p.ln == 0 {
			if p.type_ != PT_EOL && p.type_ != PT_EOF {
				p.type_ = PT_EOL
			} else {
				p.type_ = PT_EOF
			}
			return PICOL_OK
		}

		switch p.text[p.p] {
		case ' ', '\t', '\r':
			if p.insidequote != 0 {
				return p.ParseString()
			}
			return p.ParseSep()
		case '\n', ';':
			if p.insidequote != 0 {
				return p.ParseString()
			}
			return p.ParseEol()
		case '[':
			return p.ParseCommand()
		case '$':
			return p.ParseVar()
		case '#':
			if p.type_ == PT_EOL {
				p.ParseComment()
				continue
			}
			return p.ParseString()
		default:
			return p.ParseString()
		}
	}
	return PICOL_OK /* unreached */
}
