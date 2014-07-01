package picol

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

func (p *Parser) next() {
	p.p++
	p.ln--
}

func (p *Parser) token() (t string) {
	defer recover()
	return p.text[p.start:p.end]
}

func (p *Parser) parseSep() string {
	p.start = p.p
Loop:
	for p.p != len(p.text) {
		switch p.text[p.p] {
		case ' ', '\t', '\n', '\r':
		default:
			break Loop
		}
		p.next()
	}
	p.end = p.p - 1
	p.type_ = PT_SEP
	return p.token()
}

func (p *Parser) parseEol() string {
	p.start = p.p
Loop:
	for p.p != len(p.text) {
		switch p.text[p.p] {
		case ';', ' ', '\t', '\n', '\r':
		default:
			break Loop
		}
		p.next()
	}
	p.end = p.p
	p.type_ = PT_EOL
	return p.token()
}

func (p *Parser) parseCommand() string {
	level, blevel := 1, 0
	p.next() // skip
	p.start = p.p
Loop:
	for {
		switch {
		case p.ln == 0:
			break Loop
		case p.text[p.p] == '[' && blevel == 0:
			level++
		case p.text[p.p] == ']' && blevel == 0:
			level--
			if level == 0 {
				break Loop
			}
		case p.text[p.p] == '\\':
			p.next()
		case p.text[p.p] == '{':
			blevel++
		case p.text[p.p] == '}':
			if blevel != 0 {
				blevel--
			}
		}
		p.next()
	}
	p.end = p.p
	p.type_ = PT_CMD
	if p.p != len(p.text) && p.text[p.p] == ']' {
		p.next()
	}
	return p.token()
}

func (p *Parser) parseVar() string {
	p.next() // skip the $
	p.start = p.p
	for p.p != len(p.text) {
		c := p.text[p.p]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' {
			p.next()
			continue
		}
		break
	}
	if p.start == p.p { // It's just a single char string "$"
		p.start = p.p - 1
		p.end = p.p
		p.type_ = PT_STR
	} else {
		p.end = p.p
		p.type_ = PT_VAR
	}
	return p.token()
}

func (p *Parser) parseBrace() string {
	level := 1
	p.next() // skip
	p.start = p.p

Loop:
	for p.p != len(p.text) {
		c := p.text[p.p]
		switch {
		case p.ln >= 2 && c == '\\':
			p.next()
		case p.ln == 0 || c == '}':
			level--
			if level == 0 || p.ln == 0 {
				break Loop
			}
		case c == '{':
			level++
		}
		p.next()
	}
	p.end = p.p
	if p.ln != 0 { // Skip final closed brace
		p.next()
	}
	p.type_ = PT_STR
	return p.token()
}

func (p *Parser) parseString() string {
	newword := p.type_ == PT_SEP || p.type_ == PT_EOL || p.type_ == PT_STR
	if c := p.text[p.p]; newword && c == '{' {
		return p.parseBrace()
	} else if newword && c == '"' {
		p.insidequote = 1
		p.next() // skip
	}
	p.start = p.p
Loop:
	for {
		if p.ln == 0 {
			break Loop
		}
		switch p.text[p.p] {
		case '\\':
			if p.ln >= 2 {
				p.next()
			}
		case '$':
		case '[':
			break Loop
		case ' ', '\t', '\n', '\r', ';':
			if p.insidequote == 0 {
				break Loop
			}
		case '"':
			if p.insidequote != 0 {
				p.end = p.p
				p.type_ = PT_ESC
				p.next()
				p.insidequote = 0
				return p.token()
			}
		}
		p.next()
	}
	p.end = p.p
	p.type_ = PT_ESC
	return p.token() /* unreached */
}

func (p *Parser) parseComment() string {
	for p.ln != 0 && p.text[p.p] != '\n' {
		p.next()
	}
	return p.token()
}

func (p *Parser) GetToken() string {
	for {
		if p.ln == 0 {
			if p.type_ != PT_EOL && p.type_ != PT_EOF {
				p.type_ = PT_EOL
			} else {
				p.type_ = PT_EOF
			}
			return p.token()
		}

		switch p.text[p.p] {
		case ' ', '\t', '\r':
			if p.insidequote != 0 {
				return p.parseString()
			}
			return p.parseSep()
		case '\n', ';':
			if p.insidequote != 0 {
				return p.parseString()
			}
			return p.parseEol()
		case '[':
			return p.parseCommand()
		case '$':
			return p.parseVar()
		case '#':
			if p.type_ == PT_EOL {
				p.parseComment()
				continue
			}
			return p.parseString()
		default:
			return p.parseString()
		}
	}
	return p.token() /* unreached */
}
