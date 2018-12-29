package clean_html

import (
	"bytes"
	"fmt"
	"io"

	"golang.org/x/net/html"
)

type reader struct {
	s        []byte
	i        int64
	z        int64
	prevRune int64 // index of the previously read rune or -1
}

// func (r *reader) string() string {
// 	return r.s
// }

// func (r *reader) len() int {
// 	if r.i >= r.z {
// 		return 0
// 	}
// 	return int(r.z - r.i)
// }

// func (r *Reader) size() int64 {
// 	return r.z
// }

func (r *reader) pos() int64 {
	return r.i
}

func (r *reader) Read(b []byte) (int, error) {
	if r.i >= r.z {
		return 0, io.EOF
	}

	r.prevRune = -1
	b[0] = r.s[r.i]
	r.i += 1
	return 1, nil
}

func newReader(text []byte) *reader {
	return &reader{
		s: text, 
		z: int64(len(text)),
		prevRune: int64(-1),
	}
}

type Portions struct {
	Positions [][2]int 
	Bolded, Italicised []bool
	Adjusted [][2]int
}

// Need to change this to be depth based for bolding and italics ... 
func TextPos(text []byte) (Portions, error) {
	reader := newReader(text)
	tokenizer := html.NewTokenizer(reader)
	
	prev := 0
	var boldDepth, italicDepth int  

	inds := make([][2]int, 0)
	italics := make([]bool, 0)
	bolded := make([]bool, 0)
	
	loop: 
	for {
		token := tokenizer.Next()
		switch token {
		case html.ErrorToken:
			if tokenizer.Err() == io.EOF {
				break loop
			} else {
				return Portions{}, fmt.Errorf("Error: %s\n", html.ErrorToken)
			}
		case html.TextToken:
			inds = append(inds, [2]int{prev, prev+len(tokenizer.Raw())})
			italicised := false 
			if italicDepth > 0 {
				italicised = true 
			}
			italics = append(italics, italicised)
			bold := false 
			if boldDepth > 0 {
				bold = true 
			}
			bolded = append(bolded, bold)
		case html.EndTagToken:
			prev = int(reader.pos())
			tag, _ := tokenizer.TagName()
			if bytes.Equal(tag, []byte("i")) {
				italicDepth--
				if italicDepth < 0 {
					return Portions{}, fmt.Errorf("Malformed html with italic depth below 0.\n")
				}
			} else if bytes.Equal(tag, []byte("b")) {
				boldDepth--
				if boldDepth < 0 {
					return Portions{}, fmt.Errorf("Malformed html with bold depth below 0.\n")
				}
			}
		case html.StartTagToken:
			prev = int(reader.pos())
            tag, _ := tokenizer.TagName()
			if bytes.Equal(tag, []byte("i")) {
				italicDepth++
			} else if bytes.Equal(tag, []byte("b")) {
				boldDepth++
			}
		default:
			prev = int(reader.pos())
		}
	}
	return Portions{
			inds, 
			bolded,
			italics,
			nil,
		}, nil 
}

func TextPosWithCleanStyleTags(text []byte) (Portions, []byte, error) {
	reader := newReader(text)
	tokenizer := html.NewTokenizer(reader)

	prev := 0
	var boldDepth, italicDepth, adjustedPos int

	inds := make([][2]int, 0)
	adjusted := make([][2]int, 0)
	italics := make([]bool, 0)
	bolded := make([]bool, 0)

	var outBuff bytes.Buffer

loop:
	for {
		token := tokenizer.Next()
		switch token {
		case html.ErrorToken:
			if tokenizer.Err() == io.EOF {
				break loop
			} else {
				return Portions{}, nil, fmt.Errorf("Error: %s\n", html.ErrorToken)
			}
		case html.TextToken:
			outBuff.Write(tokenizer.Raw())
			l := len(tokenizer.Raw())
			inds = append(inds, [2]int{prev, prev+l})
			italicised := false
			if italicDepth > 0 {
				italicised = true
			}
			italics = append(italics, italicised)
			bold := false
			if boldDepth > 0 {
				bold = true
			}
			bolded = append(bolded, bold)
			adjusted= append(adjusted, [2]int{adjustedPos, adjustedPos+l})
			adjustedPos += l
		case html.EndTagToken:
			prev = int(reader.pos())
			tag, _ := tokenizer.TagName()
			if bytes.Equal(tag, []byte("i")) {
				italicDepth--
				if italicDepth < 0 {
					return Portions{}, nil, fmt.Errorf("Malformed html with italic depth below 0.\n")
				}
				// for </b>
				adjustedPos += 4
				outBuff.WriteString("</i>")
			} else if bytes.Equal(tag, []byte("b")) {
				boldDepth--
				if boldDepth < 0 {
					return Portions{}, nil, fmt.Errorf("Malformed html with bold depth below 0.\n")
				}
				adjustedPos += 4
				outBuff.WriteString("</b>")
			}
		case html.StartTagToken:
			prev = int(reader.pos())
			tag, _ := tokenizer.TagName()
			if bytes.Equal(tag, []byte("i")) {
				italicDepth++
				adjustedPos += 3
				outBuff.WriteString("<i>")
			} else if bytes.Equal(tag, []byte("b")) {
				boldDepth++
				adjustedPos += 4
				outBuff.WriteString("<b>")
			}
		case html.SelfClosingTagToken:
			prev = int(reader.pos())
			tag, _ := tokenizer.TagName()
			if bytes.Equal(tag, []byte("br")) {
				outBuff.WriteByte(' ')
				adjustedPos += 1
			}
		default:
			prev = int(reader.pos())
		}
	}

	return Portions{
		inds,
		bolded,
		italics,
		adjusted,
	}, outBuff.Bytes(), nil
}


func CleanText(text []byte) ([]byte, error) {
	r := bytes.NewReader(text)
	tokenizer := html.NewTokenizer(r)

	var outBuff bytes.Buffer

loop:
	for {
		token := tokenizer.Next()
		switch token {
		case html.ErrorToken:
			if tokenizer.Err() == io.EOF {
				break loop
			} else {
				return nil, fmt.Errorf("Error: %s\n", html.ErrorToken)
			}
		case html.TextToken:
			outBuff.Write(tokenizer.Raw())
		}
	}
	return outBuff.Bytes(), nil
}

func CleanTextWithStyleTags(text []byte) ([]byte, error) {
	r := bytes.NewReader(text)
	tokenizer := html.NewTokenizer(r)

	var outBuff bytes.Buffer
loop:
	for {
		token := tokenizer.Next()
		switch token {
		case html.ErrorToken:
			if tokenizer.Err() == io.EOF {
				break loop
			} else {
				return nil, fmt.Errorf("Error: %s\n", html.ErrorToken)
			}
		case html.TextToken:
			outBuff.Write(tokenizer.Raw())
		case html.EndTagToken:
			tag, _ := tokenizer.TagName()
			if bytes.Equal(tag, []byte("i")) {
				outBuff.WriteString("</i>")
			} else if bytes.Equal(tag, []byte("b")) {
				outBuff.WriteString("</b>")
			}
		case html.StartTagToken:
			tag, _ := tokenizer.TagName()
			if bytes.Equal(tag, []byte("i")) {
				outBuff.WriteString("<i>")
			} else if bytes.Equal(tag, []byte("b")) {
				outBuff.WriteString("<b>")
			}
		case html.SelfClosingTagToken:
			tag, _ := tokenizer.TagName()
			if bytes.Equal(tag, []byte("br")) {
				outBuff.WriteByte(' ')
			}
		}
	}
	return outBuff.Bytes(), nil
}

// func test(text string) {
// 	pos, err := TextPos(text)
// 	if err != nil {
// 		panic(err)
// 	}
// 	for i := range pos.Positions {
// 		fmt.Printf("Italicised: %t, bold: %t, text: %q\n", pos.Italicised[i], pos.Bolded[i], text[pos.Positions[i][0]:pos.Positions[i][1]])
// 	}
// }

// func main() {
// 	text := `Hello <i> This is a test </i>`
// 	text2 := "<b> This is a text </b>"
// 	test(text)
// 	test(text2)
// }



