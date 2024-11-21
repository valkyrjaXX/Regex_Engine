package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Split(bufio.ScanLines)

	for scanner.Scan() {
		values := strings.SplitN(scanner.Text(), "|", 2)
		regex, text := values[0], values[1]

		expression, _, _ := evalExpression(regex, text)
		fmt.Println(expression)
	}

	//file, err := os.Open("C:\\Users\\ocher\\GolandProjects\\awesomeProject\\examples.txt")
	//if err != nil {
	//	log.Fatal(err)
	//}
	//
	//scanner := bufio.NewScanner(file)
	//scanner.Split(bufio.ScanLines)
	//
	//for scanner.Scan() {
	//	text := scanner.Text()
	//	values := strings.SplitN(text, "|", 3)
	//	regex, input := values[0], values[1]
	//
	//	expression, start, end := evalExpression(regex, input)
	//	fmt.Println(fmt.Sprintf("%s|%v|%d|%d", text, expression, start, end))
	//}
}

func evalExpression(regex, input string) (bool, int, int) {
	if regex == "" || regex == string(anything) {
		return true, 0, len(input)
	}

	if input == "" {
		return false, -1, -1
	}

	lexeme := parseRegexp(regex)
	res, start, end := lexeme.Eval(input)
	return res, start, end
}

// Lexeme

const (
	anything   = '.'
	beginning  = '^'
	ending     = '$'
	zeroOrOne  = '?'
	zeroOrMore = '*'
	oneOrMore  = '+'
	escape     = '\\'
)

func parseRegexp(regex string) Lexeme {
	regexStart := regex[0]
	regexEnd := regex[len(regex)-1]

	matcherRegex := regex
	if regexStart == beginning {
		matcherRegex = matcherRegex[1:]
	}

	if regexEnd == ending {
		matcherRegex = matcherRegex[:len(matcherRegex)-1]
	}

	result := subPhraseLexeme(matcherRegex)
	if regexStart == beginning {
		result = beginningLexeme(result)
	}

	if regexEnd == ending {
		result = endingLexeme(result)
	}

	return result
}

// Lexemes

type Lexeme interface {
	Eval(string) (bool, int, int)
}

type phraseLexeme struct {
	regex    string
	evalFunc func(string) (bool, int, int)
}

func (l *phraseLexeme) Eval(s string) (bool, int, int) {
	return l.evalFunc(s)
}

type wrappingLexeme struct {
	lexeme   Lexeme
	evalFunc func(string) (bool, int, int)
}

func (l *wrappingLexeme) Eval(s string) (bool, int, int) {
	return l.evalFunc(s)
}

var beginningLexeme = func(lexeme Lexeme) Lexeme {
	return &wrappingLexeme{
		lexeme: lexeme,
		evalFunc: func(input string) (bool, int, int) {
			res, start, end := lexeme.Eval(input)
			if res && start == 0 {
				return true, start, end
			}

			return false, -1, -1
		},
	}
}

var endingLexeme = func(lexeme Lexeme) Lexeme {
	return &wrappingLexeme{
		lexeme: lexeme,
		evalFunc: func(input string) (bool, int, int) {
			res, start, end := lexeme.Eval(input)
			if res && end == len(input) {
				return true, start, end
			}

			return false, -1, -1
		},
	}
}

var subPhraseLexeme = func(regex string) Lexeme {
	return &phraseLexeme{
		regex: regex,
		evalFunc: func(input string) (bool, int, int) {
			if input == regex {
				return true, 0, len(input)
			}

			if len(regex) == 1 && regex[0] == anything {
				return true, 0, len(input)
			}

			chunks := parseRegexChunks(regex)

			rest := input
			startIdx := -1
			var endIdx int
			for _, regexChunk := range chunks {
				if len(regexChunk) == 2 && isRepetitiveSymbol(rune(regexChunk[1])) {
					regexSymbol := regexChunk[0]
					if regexSymbol != escape {
						switch regexChunk[1] {
						case zeroOrOne:
							if regexSymbol == anything || rest[0] != regexSymbol {
								continue
							}

							if len(rest) == 1 {
								endIdx += len(input) - 1
								continue
							}

							if rest[1] == regexSymbol {
								return false, -1, -1
							}

							endIdx++
							rest = rest[1:]

							continue
						case zeroOrMore:
							if regexSymbol == anything || rest[0] != regexSymbol {
								if startIdx == -1 {
									startIdx = 0
								}
								continue
							}

							if len(rest) == 1 {
								endIdx += len(input) - 1
								continue
							}

							for idx, inputChar := range rest {
								if inputChar != rune(regexSymbol) {
									endIdx += idx
									rest = rest[idx:]
									break
								}
							}

							continue

						case oneOrMore:
							if regexSymbol == anything {
								continue
							}

							if rest[0] != regexSymbol {
								return false, -1, -1
							}

							if len(rest) == 1 {
								endIdx += len(input) - 1
								continue
							}

							for idx, inputChar := range rest {
								if inputChar != rune(regexSymbol) {
									endIdx += idx
									rest = rest[idx:]
									break
								}
							}

							continue
						}
					}
				}

				var match bool
				localRest := rest

				var inputStartIdx int

				for {
					if inputStartIdx >= len(localRest) {
						break
					}

					inputEndIdx := inputStartIdx + len(regexChunk) - escapeSymbolsNum(regexChunk)
					if inputEndIdx > len(localRest) {
						inputEndIdx = len(localRest)
					}

					inputChunk := localRest[inputStartIdx:inputEndIdx]
					if matchSubstring(regexChunk, inputChunk) {
						if startIdx == -1 {
							startIdx = inputStartIdx
						}

						if match {
							endIdx = inputEndIdx
						} else {
							endIdx += inputEndIdx
						}

						if endIdx < len(input) {
							rest = rest[endIdx:]
						}

						match = true
						inputStartIdx += len(inputChunk)
						match = true
						continue
					}

					inputStartIdx++
				}

				if !match {
					return false, -1, -1
				}
			}

			return true, startIdx, endIdx
		},
	}
}

func matchSubstring(regex, input string) bool {
	var regexIdx int
	var inputIdx int

	var escapeLastKnownIndex = -1
	for {
		if inputIdx == len(input) {
			return true
		}

		regExChar := rune(regex[regexIdx])
		if regExChar == escape && (escapeLastKnownIndex == -1 || escapeLastKnownIndex != regexIdx-1) {
			escapeLastKnownIndex = regexIdx
			regexIdx++
			continue
		}

		if regexIdx == 0 {
			if regExChar != anything && regExChar != rune(input[inputIdx]) {
				return false
			}

			regexIdx++
			inputIdx++
			continue
		}

		prev := regex[regexIdx-1]
		if prev == escape && regExChar != rune(input[inputIdx]) {
			return false
		}

		if regExChar != anything && regExChar != rune(input[inputIdx]) {
			return false
		}

		regexIdx++
		inputIdx++
	}
}

func parseRegexChunks(regex string) []string {
	var chunks []string

	var idx int

	var chunk string
	for {
		if idx == len(regex) {
			if len(chunk) > 0 {
				chunks = append(chunks, chunk)
			}

			return chunks
		}

		currentChar := rune(regex[idx])
		if isEscape(currentChar) {
			chunk += regex[idx : idx+2]
			idx += 2
			continue
		}

		if isRepetitiveSymbol(currentChar) {
			sub := chunk[:len(chunk)-1]
			if sub != "" {
				chunks = append(chunks, sub)
				chunk = ""
			}

			chunks = append(chunks, regex[idx-1:idx+1])

			if len(regex) > idx+1 {
				chunks = append(chunks, parseRegexChunks(regex[idx+1:])...)
			}

			return chunks
		}

		chunk += string(regex[idx])
		idx++
	}
}

func escapeSymbolsNum(regex string) int {
	var count int
	var lastKnownIndex = -1
	for idx, c := range regex {
		if c == escape && lastKnownIndex != idx-1 {
			lastKnownIndex = idx
			count++
		}
	}

	return count
}

func isEscape(c rune) bool {
	return c == escape
}

func isRepetitiveSymbol(c rune) bool {
	return c == zeroOrOne || c == zeroOrMore || c == oneOrMore
}
