package utils

func H1(text string) string {
	return "# " + text
}

func H2(text string) string {
	return "## " + text
}

func H3(text string) string {
	return "### " + text
}

func H4(text string) string {
	return "#### " + text
}

func Bold(text string) string {
	return "**" + text + "**"
}

func Italic(text string) string {
	return "_" + text + "_"
}

func InlineCode(text string) string {
	return "`" + text + "`"
}

func Link(text, url string) string {
	return "[" + text + "](" + url + ")"
}
